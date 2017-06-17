package app

import (
	"acceptance/config"
	"acceptance/helpers"
	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var _ = Describe("Dynamic scaling based on memory metrics", func() {
	var (
		appName              string
		appGUID              string
		instanceName         string
		initialInstanceCount int
		policy               string
	)

	BeforeEach(func() {
		instanceName = generator.PrefixedRandomName("autoscaler", "service")
		createService := cf.Cf("create-service", cfg.ServiceName, cfg.ServicePlan, instanceName).Wait(cfg.DefaultTimeoutDuration())
		Expect(createService).To(Exit(0), "failed creating service")
	})

	JustBeforeEach(func() {
		appName = generator.PrefixedRandomName("autoscaler", "nodeapp")
		countStr := strconv.Itoa(initialInstanceCount)
		createApp := cf.Cf("push", appName, "--no-start", "-i", countStr, "-b", cfg.NodejsBuildpackName, "-m", "128M", "-p", config.NODE_APP, "-d", cfg.AppsDomain).Wait(cfg.DefaultTimeoutDuration())
		Expect(createApp).To(Exit(0), "failed creating app")

		guid := cf.Cf("app", appName, "--guid").Wait(cfg.DefaultTimeoutDuration())
		Expect(guid).To(Exit(0))
		appGUID = strings.TrimSpace(string(guid.Out.Contents()))

		Expect(cf.Cf("start", appName).Wait(cfg.CfPushTimeoutDuration())).To(Exit(0))
		waitForNInstancesRunning(appGUID, initialInstanceCount, cfg.DefaultTimeoutDuration())

		bindService := cf.Cf("bind-service", appName, instanceName, "-c", policy).Wait(cfg.DefaultTimeoutDuration())
		Expect(bindService).To(Exit(0), "failed binding service to app with a policy ")

	})

	AfterEach(func() {
		unbindService := cf.Cf("unbind-service", appName, instanceName).Wait(cfg.DefaultTimeoutDuration())
		Expect(unbindService).To(Exit(0), "failed unbinding service from app")

		Expect(cf.Cf("delete", appName, "-f", "-r").Wait(cfg.DefaultTimeoutDuration())).To(Exit(0))
		Expect(cf.Cf("delete-service", instanceName, "-f").Wait(cfg.DefaultTimeoutDuration())).To(Exit(0))

	})

	Context("When scaling by throughput", func() {
		Context("when througput is great than given scaling out threshold", func() {
			BeforeEach(func() {
				initialInstanceCount = 1
				policy = generateDynamicScaleOutPolicy(1, 2, "throughput", 1)
			})
			JustBeforeEach(func() {
				startStress(appName, 200)
			})
			AfterEach(func() {
				stopStress()
			})
			It("should scale out", func() {
				totalTime := time.Duration(interval*2)*time.Second + 2*time.Minute
				finishTime := time.Now().Add(totalTime)
				waitForNInstancesRunning(appGUID, 2, finishTime.Sub(time.Now()))
			})
		})
	})
})

var doneChannel chan bool

func startStress(name string, sleepInterval int) {
	url := fmt.Sprintf("http://%s.%s?inteval=%d", name, cfg.GetAppsDomain(), sleepInterval)
	doneChannel = make(chan bool)
	go func() {
		num := 0
		for {
			select {
			case <-doneChannel:
				return
			default:
				status, _, err := helpers.Curl(cfg, "-s", url)
				Expect(err).NotTo(HaveOccurred())
				Expect(status).To(Equal(http.StatusOK))
				num++
				fmt.Printf("%d %s\n", num, url)
			}
		}
	}()
}

func stopStress() {
	close(doneChannel)
}
