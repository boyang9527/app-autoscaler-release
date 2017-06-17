var url = require('url');
var http = require('http');
var sleep = require('sleep');


var server = http.createServer(function handler(req, res) {
	var interval = url.parse(req.url, true).query.interval;	
	if  ( !interval || (interval < 0) ) {
		interval = 0 
	}
	if (interval > 0 ) {
		sleep.msleep(interval)		
	}
	res.end('sleep for ' + interval + ' ms\n')
}).listen(process.env.PORT || 8080);

console.log('App listening on port 8080');
