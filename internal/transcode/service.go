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

const (
	defaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64)"
	defaultReferer   = "https://www.youtube.com/"
)

type ffmpegInput struct {
	args     []string
	pipe     bool
	postSeek bool
	start    float64
	srcURL   string
}

// Service converts modern streams to legacy-friendly formats on the fly.
type Service struct {
	command     string
	client      *http.Client
	resolver    StreamResolver
	rtsp        *rtspServer
	rtspAddr    string
	retroFilter string
}

const DefaultRetroFilter = "eq=contrast=1.08:saturation=1.08,unsharp=4:4:0.45"

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

// WithRetroFilter appends an extra filter chain for 3GP retro/edge outputs.
func (s *Service) WithRetroFilter(filter string) *Service {
	s.retroFilter = strings.TrimSpace(filter)
	return s
}

func (s *Service) buildInput(srcURL string, start float64) (ffmpegInput, func(), error) {
	spec := ffmpegInput{
		args:   []string{"-hide_banner", "-re"},
		srcURL: srcURL,
		start:  start,
	}
	var cleanup func()

	if requiresPipe(srcURL) {
		proxy, err := newStreamProxy(s.client, srcURL)
		if err == nil {
			srcURL = proxy.URL()
			cleanup = proxy.Close
		} else {
			log.Printf("[proxy] falling back to pipe input: %v", err)
			spec.pipe = true
			spec.args = append(spec.args, "-i", "pipe:0")
			if start > 0 {
				spec.postSeek = true
			}
			return spec, nil, nil
		}
	}

	if start > 0 {
		spec.args = append(spec.args, "-ss", formatSeek(start))
	}
	spec.args = append(spec.args,
		"-headers", fmt.Sprintf("Referer: %s\r\n", defaultReferer),
		"-user_agent", defaultUserAgent,
		"-i", srcURL,
	)
	spec.srcURL = srcURL
	return spec, cleanup, nil
}

// Stream launches ffmpeg and proxies the converted output to the ResponseWriter.
func (s *Service) Stream(ctx context.Context, w http.ResponseWriter, srcURL, videoID string, profile Profile, start float64) error {
	input, cleanup, err := s.buildInput(srcURL, start)
	if err != nil {
		return err
	}
	if cleanup != nil {
		defer cleanup()
	}

	args, format, err := s.profileArgs(profile, input)
	if err != nil {
		return err
	}

	cmd := exec.CommandContext(ctx, s.command, args...)
	var stdin io.WriteCloser
	if input.pipe {
		stdInPipe, err := cmd.StdinPipe()
		if err != nil {
			return fmt.Errorf("stdin pipe: %w", err)
		}
		stdin = stdInPipe
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
		if stdin != nil {
			_ = stdin.Close()
		}
		return fmt.Errorf("ffmpeg start: %w", err)
	}

	if input.pipe && stdin != nil {
		s.startInputPump(ctx, stdin, input.srcURL)
	}

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

func (s *Service) profileArgs(profile Profile, input ffmpegInput) (args []string, format outputFormat, err error) {
	args = append(args, input.args...)
	if input.pipe && input.postSeek {
		args = append(args, "-ss", formatSeek(input.start))
	}

	switch profile {
	case ProfileAAC:
		args = append(args,
			"-vf", "scale=320:240,fps=15",
			"-c:v", "mpeg4", "-b:v", "256k",
			"-c:a", "aac", "-ar", "16000", "-ac", "1", "-b:a", "32k",
			"-movflags", "frag_keyframe+empty_moov",
			"-f", "mp4", "pipe:1",
		)
		return args, outputFormat{ContentType: "video/mp4", Extension: "mp4", Suffix: "_aac"}, nil
	case ProfileMP3:
		args = append(args,
			"-vf", "scale=176:144,fps=12",
			"-c:v", "mpeg4", "-b:v", "120k",
			"-c:a", "libmp3lame", "-ar", "11025", "-ac", "1", "-b:a", "24k",
			"-f", "avi", "pipe:1",
		)
		return args, outputFormat{ContentType: "video/x-msvideo", Extension: "avi", Suffix: "_mp3"}, nil
	case ProfileEdge:
		vf := buildFilterChain([]string{"scale=128:96", "fps=10"}, s.retroFilter)
		args = append(args,
			"-vf", vf,
			"-c:v", "h263", "-b:v", "60k",
			"-c:a", "libopencore_amrnb", "-ar", "8000", "-ac", "1", "-b:a", "10.2k",
			"-use_editlist", "0",
			"-movflags", "+faststart+frag_keyframe+empty_moov",
			"-f", "3gp", "pipe:1",
		)
		return args, outputFormat{ContentType: "video/3gpp", Extension: "3gp", Suffix: "_edge"}, nil
	case ProfileRetro, "":
		vf := buildFilterChain([]string{"scale=176:144", "fps=12"}, s.retroFilter)
		args = append(args,
			"-vf", vf,
			"-c:v", "h263", "-b:v", "120k",
			"-c:a", "libopencore_amrnb", "-ar", "8000", "-ac", "1", "-b:a", "12.2k",
			"-use_editlist", "0",
			"-movflags", "+faststart+frag_keyframe+empty_moov",
			"-f", "3gp", "pipe:1",
		)
		return args, outputFormat{ContentType: "video/3gpp", Extension: "3gp", Suffix: "_retro"}, nil
	default:
		return nil, outputFormat{}, fmt.Errorf("unknown profile %q", profile)
	}
}

func (s *Service) profileRTSPArgs(profile Profile, input ffmpegInput, target string) ([]string, error) {
	base := append([]string{}, input.args...)
	if input.pipe && input.postSeek {
		base = append(base, "-ss", formatSeek(input.start))
	}

	switch profile {
	case ProfileRetro, "":
		vf := buildFilterChain([]string{"scale=176:144", "fps=12"}, s.retroFilter)
		base = append(base,
			"-vf", vf,
			"-c:v", "h263", "-b:v", "120k",
			"-c:a", "aac", "-ar", "32000", "-ac", "1", "-b:a", "32k",
		)
	case ProfileEdge:
		vf := buildFilterChain([]string{"scale=128:96", "fps=10"}, s.retroFilter)
		base = append(base,
			"-vf", vf,
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

func buildFilterChain(base []string, extra string) string {
	chain := strings.Join(base, ",")
	extra = strings.Trim(extra, " ,")
	if extra != "" {
		if chain != "" {
			chain += ","
		}
		chain += extra
	}
	return chain
}

func formatSeek(value float64) string {
	if value < 0 {
		value = 0
	}
	return fmt.Sprintf("%.3f", value)
}

func requiresPipe(src string) bool {
	src = strings.ToLower(strings.TrimSpace(src))
	return strings.HasPrefix(src, "https://")
}

func (s *Service) startInputPump(ctx context.Context, dst io.WriteCloser, src string) {
	go func() {
		defer dst.Close()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, src, nil)
		if err != nil {
			log.Printf("[fetch] request build: %v", err)
			return
		}
		req.Header.Set("User-Agent", defaultUserAgent)
		req.Header.Set("Referer", defaultReferer)

		resp, err := s.client.Do(req)
		if err != nil {
			log.Printf("[fetch] %v", err)
			return
		}
		defer resp.Body.Close()

		if _, err := io.Copy(dst, resp.Body); err != nil {
			if ctx.Err() == nil {
				log.Printf("[fetch-copy] %v", err)
			}
		}
	}()
}
