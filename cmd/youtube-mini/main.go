package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"youtube-mini/internal/app"
	"youtube-mini/internal/transcode"
	"youtube-mini/internal/youtube"
)

const (
	defaultAddr   = ":8090"
	defaultAPIKey = "AIzaSyAO_FJ2SlqU8Q4STEHLGCilw_Y9_11qcW8"
)

func main() {
	startAdmin()

	addr := getenv("YOUTUBE_MINI_ADDR", defaultAddr)
	apiKey := getenv("YOUTUBE_API_KEY", defaultAPIKey)
	rtspAddr := getenv("YTM_RTSP_ADDR", "")

	yt := youtube.New(apiKey)
	legacy := transcode.New()

	retroFilterEnv := strings.TrimSpace(os.Getenv("YTM_RETRO_FILTER"))
	switch strings.ToLower(retroFilterEnv) {
	case "", "off", "false", "0", "disable":
		// noop
	case "default":
		legacy.WithRetroFilter(transcode.DefaultRetroFilter)
	default:
		legacy.WithRetroFilter(retroFilterEnv)
	}

	if transport := strings.TrimSpace(os.Getenv("YTM_RTSP_TRANSPORT")); transport != "" {
		legacy.WithRTSPTransport(transport)
	}
	rtpEnv := strings.TrimSpace(os.Getenv("YTM_RTSP_UDP_RTP"))
	rtcpEnv := strings.TrimSpace(os.Getenv("YTM_RTSP_UDP_RTCP"))
	if rtpEnv != "" || rtcpEnv != "" {
		legacy.WithRTSPUDPPorts(rtpEnv, rtcpEnv)
	}

	legacy.WithStreamResolver(transcode.StreamResolverFunc(func(ctx context.Context, videoID string) (string, error) {
		video, err := yt.GetVideo(ctx, videoID)
		if err != nil {
			return "", err
		}
		if video.Stream == "" {
			return "", fmt.Errorf("stream not available")
		}
		return video.Stream, nil
	}))
	if err := legacy.EnableRTSP(rtspAddr); err != nil {
		log.Fatalf("rtsp: %v", err)
	}

	server := app.New(yt, legacy)

	srv := &http.Server{
		Addr:         addr,
		Handler:      server.Handler(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 0,
	}

	log.Printf("YouTube Mini Retro listening on %s", addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
