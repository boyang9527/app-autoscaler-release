package app

import (
	"acceptance/config"
	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"fmt"
	"strconv"
	"strings"
	"time"
)

var _ = Describe("AutoScaler recurring schedule policy", func() {
	var (
		appName              string
		appGUID              string
		instanceName         string
		initialInstanceCount int
		daysOfMonthOrWeek    Days
		startTime            time.Time
		endTime              time.Time
	)

	BeforeEach(func() {
		instanceName = generator.PrefixedRandomName("autoscaler", "service")
		createService := cf.Cf("create-service", cfg.ServiceName, cfg.ServicePlan, instanceName).Wait(cfg.DefaultTimeoutDuration())
		Expect(createService).To(Exit(0), "failed creating service")

		initialInstanceCount = 1
		appName = generator.PrefixedRandomName("autoscaler", "nodeapp")
		countStr := strconv.Itoa(initialInstanceCount)
		createApp := cf.Cf("push", appName, "--no-start", "-i", countStr, "-b", cfg.NodejsBuildpackName, "-m", fmt.Sprintf("%dM", cfg.NodeMemoryLimit), "-p", config.NODE_APP, "-d", cfg.AppsDomain).Wait(cfg.DefaultTimeoutDuration())
		Expect(createApp).To(Exit(0), "failed creating app")

		guid := cf.Cf("app", appName, "--guid").Wait(cfg.DefaultTimeoutDuration())
		Expect(guid).To(Exit(0))
		appGUID = strings.TrimSpace(string(guid.Out.Contents()))
	})

	AfterEach(func() {
		Expect(cf.Cf("delete", appName, "-f", "-r").Wait(cfg.DefaultTimeoutDuration())).To(Exit(0))
		Expect(cf.Cf("delete-service", instanceName, "-f").Wait(cfg.DefaultTimeoutDuration())).To(Exit(0))
	})

	Context("when scale out by recurring schedule", func() {

		JustBeforeEach(func() {
			location, err := time.LoadLocation("GMT")
			Expect(err).NotTo(HaveOccurred())
			startTime, endTime = getStartAndEndTime(location, 70*time.Second, 4*time.Minute)
			policyStr := generateDynamicAndRecurringSchedulePolicy(1, 4, 80, "GMT", startTime, endTime, daysOfMonthOrWeek, 2, 5, 3)

			bindService := cf.Cf("bind-service", appName, instanceName, "-c", policyStr).Wait(cfg.DefaultTimeoutDuration())
			Expect(bindService).To(Exit(0), "failed binding service to app with a policy ")

			Expect(cf.Cf("start", appName).Wait(cfg.DefaultTimeout * 3)).To(Exit(0))
			waitForNInstancesRunning(appGUID, initialInstanceCount, cfg.DefaultTimeoutDuration())
		})

		AfterEach(func() {
			unbindService := cf.Cf("unbind-service", appName, instanceName).Wait(cfg.DefaultTimeoutDuration())
			Expect(unbindService).To(Exit(0), "failed unbinding service from app")
		})

		Context("with days of month", func() {
			BeforeEach(func() {
				daysOfMonthOrWeek = daysOfMonth
			})

			It("should scale", func() {
				waitTime := startTime.Sub(time.Now()) + 1*time.Minute
				By("setting to initial_min_instance_count")
				waitForNInstancesRunning(appGUID, 3, waitTime)

				By("setting schedule's instance_min_count")
				jobRunTime := endTime.Sub(time.Now())
				Eventually(func() int {
					return runningInstances(appGUID, jobRunTime)
				}, jobRunTime, 15*time.Second).Should(Equal(2))

				jobRunTime = endTime.Sub(time.Now())
				Consistently(func() int {
					return runningInstances(appGUID, jobRunTime)
				}, jobRunTime, 15*time.Second).Should(Equal(2))

				By("setting to default instance_min_count")
				waitForNInstancesRunning(appGUID, 1, 1*time.Minute)
			})

		})

		Context("with days of week", func() {
			BeforeEach(func() {
				daysOfMonthOrWeek = daysOfWeek
			})

			It("should scale", func() {
				waitTime := startTime.Sub(time.Now()) + 1*time.Minute

				By("setting to initial_min_instance_count")
				waitForNInstancesRunning(appGUID, 3, waitTime)

				By("setting schedule's instance_min_count")
				jobRunTime := endTime.Sub(time.Now())
				Eventually(func() int {
					return runningInstances(appGUID, jobRunTime)
				}, jobRunTime, 15*time.Second).Should(Equal(2))

				jobRunTime = endTime.Sub(time.Now())
				Consistently(func() int {
					return runningInstances(appGUID, jobRunTime)
				}, jobRunTime, 15*time.Second).Should(Equal(2))

				By("setting to default instance_min_count")
				waitForNInstancesRunning(appGUID, 1, 1*time.Minute)
			})
		})
	})

})
