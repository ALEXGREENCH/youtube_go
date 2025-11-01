package transcode

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
	"strings"

	"youtube-mini/internal/ui"
)

// Profile type enumerates ffmpeg presets we expose.
type Profile string

const (
	ProfileRetro Profile = "retro"
	ProfileAAC   Profile = "aac"
	ProfileMP3   Profile = "mp3"
	ProfileEdge  Profile = "edge"
)

// Service converts modern streams to legacy-friendly formats on the fly.
type Service struct {
	command  string
	client   *http.Client
	resolver StreamResolver
	rtsp     *rtspServer
	rtspAddr string
}

// New returns a Service with defaults.
func New() *Service {
	return &Service{
		command:  "ffmpeg",
		client:   http.DefaultClient,
		rtspAddr: defaultRTSPAddress,
	}
}

// WithCommand overrides the ffmpeg binary path.
func (s *Service) WithCommand(path string) *Service {
	s.command = path
	return s
}

// WithHTTPClient overrides the fetch client.
func (s *Service) WithHTTPClient(client *http.Client) *Service {
	if client != nil {
		s.client = client
	}
	return s
}

// Stream launches ffmpeg and proxies the converted output to the ResponseWriter.
func (s *Service) Stream(ctx context.Context, w http.ResponseWriter, srcURL, videoID string, profile Profile) error {
	args, format, err := profileArgs(profile)
	if err != nil {
		return err
	}

	cmd := exec.CommandContext(ctx, s.command, args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("stderr pipe: %w", err)
	}

	go func() {
		Scanner := newStderrScanner(stderr)
		for Scanner.Scan() {
			line := Scanner.Text()
			if !strings.Contains(line, "frame=") {
				log.Printf("[ffmpeg] %s", line)
			}
		}
	}()

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("ffmpeg start: %w", err)
	}

	go func() {
		defer stdin.Close()
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srcURL, nil)
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")
		req.Header.Set("Referer", "https://www.youtube.com/")

		resp, err := s.client.Do(req)
		if err != nil {
			log.Printf("[fetch] %v", err)
			return
		}
		defer resp.Body.Close()

		if _, err := io.Copy(stdin, resp.Body); err != nil {
			log.Printf("[fetch-copy] %v", err)
		}
	}()

	w.Header().Set("Content-Type", format.ContentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, ui.Escape(format.FileName(videoID))))
	w.Header().Set("Transfer-Encoding", "chunked")

	if _, err := io.Copy(w, stdout); err != nil {
		return fmt.Errorf("proxy: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("ffmpeg wait: %w", err)
	}
	log.Printf("[ffmpeg] finished id=%s profile=%s format=%s", videoID, profile, format.Extension)
	return nil
}

func profileArgs(profile Profile) (args []string, format outputFormat, err error) {
	switch profile {
	case ProfileAAC:
		return []string{
			"-hide_banner", "-re",
			"-i", "pipe:0",
			"-vf", "scale=320:240,fps=15",
			"-c:v", "mpeg4", "-b:v", "256k",
			"-c:a", "aac", "-ar", "16000", "-ac", "1", "-b:a", "32k",
			"-movflags", "frag_keyframe+empty_moov",
			"-f", "mp4", "pipe:1",
		}, outputFormat{ContentType: "video/mp4", Extension: "mp4", Suffix: "_aac"}, nil
	case ProfileMP3:
		return []string{
			"-hide_banner", "-re",
			"-i", "pipe:0",
			"-vf", "scale=176:144,fps=12",
			"-c:v", "mpeg4", "-b:v", "120k",
			"-c:a", "libmp3lame", "-ar", "11025", "-ac", "1", "-b:a", "24k",
			"-f", "avi", "pipe:1",
		}, outputFormat{ContentType: "video/x-msvideo", Extension: "avi", Suffix: "_mp3"}, nil
	case ProfileEdge:
		return []string{
			"-hide_banner", "-re",
			"-i", "pipe:0",
			"-vf", "scale=128:96,fps=10",
			"-c:v", "h263", "-b:v", "60k",
			"-c:a", "libopencore_amrnb", "-ar", "8000", "-ac", "1", "-b:a", "10.2k",
			"-use_editlist", "0",
			"-movflags", "+faststart+frag_keyframe+empty_moov",
			"-f", "3gp", "pipe:1",
		}, outputFormat{ContentType: "video/3gpp", Extension: "3gp", Suffix: "_edge"}, nil
	case ProfileRetro, "":
		return []string{
			"-hide_banner", "-re",
			"-i", "pipe:0",
			"-vf", "scale=176:144,fps=12",
			"-c:v", "h263", "-b:v", "120k",
			"-c:a", "libopencore_amrnb", "-ar", "8000", "-ac", "1", "-b:a", "12.2k",
			"-use_editlist", "0",
			"-movflags", "+faststart+frag_keyframe+empty_moov",
			"-f", "3gp", "pipe:1",
		}, outputFormat{ContentType: "video/3gpp", Extension: "3gp", Suffix: "_retro"}, nil
	default:
		return nil, outputFormat{}, fmt.Errorf("unknown profile %q", profile)
	}
}

func profileRTSPArgs(profile Profile, target string) ([]string, error) {
	base := []string{
		"-hide_banner", "-re",
		"-i", "pipe:0",
	}

	switch profile {
	case ProfileRetro, "":
		base = append(base,
			"-vf", "scale=176:144,fps=12",
			"-c:v", "h263", "-b:v", "120k",
			"-c:a", "libopencore_amrnb", "-ar", "8000", "-ac", "1", "-b:a", "12.2k",
		)
	case ProfileEdge:
		base = append(base,
			"-vf", "scale=128:96,fps=10",
			"-c:v", "h263", "-b:v", "60k",
			"-c:a", "libopencore_amrnb", "-ar", "8000", "-ac", "1", "-b:a", "10.2k",
		)
	default:
		return nil, fmt.Errorf("profile %s does not support RTSP output", profile)
	}

	base = append(base,
		"-f", "rtsp",
		"-rtsp_transport", "tcp",
		"-muxdelay", "0.1",
		target,
	)

	return base, nil
}

type outputFormat struct {
	ContentType string
	Extension   string
	Suffix      string
}

func (o outputFormat) FileName(videoID string) string {
	return fmt.Sprintf("%s%s.%s", videoID, o.Suffix, o.Extension)
}

func newStderrScanner(r io.Reader) *bufio.Scanner {
	scanner := bufio.NewScanner(r)
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, 1024*1024)
	return scanner
}
