package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"sync/atomic"
)

var running int32 = 1 // 1 = on, 0 = off
var logBuffer = make([]string, 0, 1000)

func adminLog(s string) {
	if len(logBuffer) > 1000 {
		logBuffer = logBuffer[1:]
	}
	logBuffer = append(logBuffer, s)
}

func startAdmin() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if !checkAuth(r) {
			requireAuth(w)
			return
		}
		
		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		fmt.Fprintf(w, `
		<!DOCTYPE html>
		<html>
		<head>
		<meta charset="utf-8">
		<title>YouTube Mini Admin</title>
		<style>
		body {
			font-family: Arial, sans-serif;
			background: #f1f1f1;
			margin: 0;
		}
		
		.header {
			background: #2c2c2c;
			color: #fff;
			padding: 10px;
			font-size: 20px;
			font-weight: bold;
		}
		
		.container {
			padding: 15px;
		}
		
		.status {
			font-size: 18px;
			margin-bottom: 15px;
		}
		
		.btn {
			display: inline-block;
			background: #cc181e;
			color: #fff;
			padding: 8px 14px;
			text-decoration: none;
			margin-right: 10px;
			border-radius: 4px;
		}
		
		.btn.green { background: #2ba640; }
		.btn.gray { background: #555; }
		
		.log-box {
			background: #111;
			color: #0f0;
			padding: 10px;
			margin-top: 15px;
			height: 300px;
			overflow-y: scroll;
			font-family: Consolas, monospace;
			font-size: 12px;
		}
		</style>
		</head>
		<body>
		
		<div class="header">YouTube Mini — Admin Console</div>
		
		<div class="container">
			<div class="status">Status: <b>%s</b></div>
		
			<a class="btn green" href="/restart">Restart</a>
			<a class="btn gray" href="/stop">Stop</a>
			<a class="btn" href="/logs">Logs</a>
		</div>
		
		</body>
		</html>
		`, status())

	})

	http.HandleFunc("/restart", func(w http.ResponseWriter, r *http.Request) {
		adminLog("Restart triggered")
		go restartApp()
		fmt.Fprintf(w, "Restarting...")
	})

	http.HandleFunc("/stop", func(w http.ResponseWriter, r *http.Request) {
		adminLog("Stop triggered")
		atomic.StoreInt32(&running, 0)
		fmt.Fprintf(w, "Stopped")
		os.Exit(0)
	})

	http.HandleFunc("/logs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		fmt.Fprint(w, `
		<html><body style="margin:0;font-family:monospace;background:#111;color:#0f0;">
		<div class="header" style="background:#2c2c2c;color:white;padding:10px;font-weight:bold;">Logs</div>
		<pre style="padding:10px;">`)
		for _, line := range logBuffer {
			fmt.Fprintf(w, "%s\n", line)
		}
		fmt.Fprint(w, "</pre></body></html>")

	})

	go func() {
		log.Println("Admin panel listening on :9090")
		http.ListenAndServe(":9090", nil)
	}()
}

func status() string {
	if atomic.LoadInt32(&running) == 1 {
		return "🟢 running"
	}
	return "🔴 stopped"
}

func restartApp() {
	cmd := exec.Command(os.Args[0])
	err := cmd.Start()
	if err != nil {
		adminLog("Restart failed: " + err.Error())
	}
	os.Exit(0)
}
