package transcode

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bluenviron/gortsplib/v4"
	"github.com/bluenviron/gortsplib/v4/pkg/base"
	"github.com/bluenviron/gortsplib/v4/pkg/description"
	"github.com/bluenviron/gortsplib/v4/pkg/format"
	"github.com/pion/rtp"
)

const (
	defaultRTSPAddress      = ":8554"
	rtspPublisherTimeout    = 12 * time.Second
	rtspResolveTimeout      = 15 * time.Second
	rtspIngestRetryInterval = 2 * time.Second
)

// StreamResolver resolves a YouTube video ID into a direct stream URL.
type StreamResolver interface {
	ResolveStream(ctx context.Context, videoID string) (string, error)
}

// StreamResolverFunc is an adapter to allow the use of regular functions as resolvers.
type StreamResolverFunc func(ctx context.Context, videoID string) (string, error)

// ResolveStream implements StreamResolver.
func (fn StreamResolverFunc) ResolveStream(ctx context.Context, videoID string) (string, error) {
	return fn(ctx, videoID)
}

// WithStreamResolver injects a video stream resolver used by the RTSP server.
func (s *Service) WithStreamResolver(res StreamResolver) *Service {
	s.resolver = res
	return s
}

// EnableRTSP spins up the internal RTSP server if not already running.
func (s *Service) EnableRTSP(addr string) error {
	if s.resolver == nil {
		return errors.New("rtsp: stream resolver not configured")
	}
	if addr == "" {
		addr = defaultRTSPAddress
	}
	if s.rtsp != nil {
		// If address changes, restart server.
		if s.rtsp.addr == addr {
			return nil
		}
		s.rtsp.close()
		s.rtsp = nil
	}

	server, err := newRTSPServer(s, addr)
	if err != nil {
		return err
	}
	s.rtsp = server
	s.rtspAddr = addr
	return nil
}

// RTSPEnabled reports whether RTSP streaming is available.
func (s *Service) RTSPEnabled() bool {
	return s.rtsp != nil
}

// RTSPURL returns the public RTSP URL for a given profile and video ID.
func (s *Service) RTSPURL(host string, profile Profile, videoID string) string {
	if s.rtsp == nil {
		return ""
	}
	return s.rtsp.publicURL(host, profile, videoID)
}

func (s *Service) resolveStream(ctx context.Context, videoID string) (string, error) {
	if s.resolver == nil {
		return "", errors.New("rtsp: stream resolver not configured")
	}
	return s.resolver.ResolveStream(ctx, videoID)
}

type rtspServer struct {
	svc        *Service
	addr       string
	port       int
	server     *gortsplib.Server
	mu         sync.Mutex
	streams    map[string]*rtspStream
	publishers map[*gortsplib.ServerSession]*rtspStream
}

func newRTSPServer(svc *Service, addr string) (*rtspServer, error) {
	rs := &rtspServer{
		svc:        svc,
		addr:       addr,
		streams:    make(map[string]*rtspStream),
		publishers: make(map[*gortsplib.ServerSession]*rtspStream),
	}

	rtspSrv := &gortsplib.Server{
		Handler:      rs,
		RTSPAddress:  addr,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	if err := rtspSrv.Start(); err != nil {
		return nil, fmt.Errorf("rtsp: start: %w", err)
	}

	rs.server = rtspSrv
	rs.port = extractPort(addr)
	log.Printf("[rtsp] listening on %s", addr)
	return rs, nil
}

func (r *rtspServer) close() {
	if r.server != nil {
		r.server.Close()
	}
}

func extractPort(address string) int {
	if address == "" {
		return 0
	}
	if !strings.Contains(address, ":") {
		return 0
	}
	_, portStr, err := net.SplitHostPort(address)
	if err != nil {
		if strings.HasPrefix(address, ":") {
			portStr = strings.TrimPrefix(address, ":")
		} else {
			return 0
		}
	}
	if portStr == "" {
		return 0
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return 0
	}
	return port
}

func (r *rtspServer) publicURL(host string, profile Profile, videoID string) string {
	if host == "" {
		host = "localhost"
	}
	host = stripPort(host)
	path := r.pathFor(profile, videoID)
	if r.port > 0 {
		return fmt.Sprintf("rtsp://%s:%d/%s", host, r.port, path)
	}
	return fmt.Sprintf("rtsp://%s/%s", host, path)
}

func stripPort(hostport string) string {
	if hostport == "" {
		return hostport
	}
	if strings.HasPrefix(hostport, "[") && strings.Contains(hostport, "]") {
		idx := strings.LastIndex(hostport, "]")
		return hostport[:idx+1]
	}
	if strings.Count(hostport, ":") == 0 {
		return hostport
	}
	host, _, err := net.SplitHostPort(hostport)
	if err != nil {
		return hostport
	}
	if host == "" {
		return "localhost"
	}
	return host
}

func (r *rtspServer) pathFor(profile Profile, videoID string) string {
	return fmt.Sprintf("%s/%s.3gp", profileSegment(profile), videoID)
}

func profileSegment(profile Profile) string {
	switch profile {
	case ProfileEdge:
		return "edge"
	case ProfileRetro, "":
		return "retro"
	default:
		return string(profile)
	}
}

func (r *rtspServer) parsePath(path string) (Profile, string, error) {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return "", "", errors.New("empty path")
	}
	parts := strings.Split(trimmed, "/")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("invalid path %q", path)
	}
	profilePart := strings.ToLower(parts[0])
	videoPart := strings.Join(parts[1:], "/")
	videoPart = strings.TrimSuffix(videoPart, ".3gp")
	videoPart = strings.TrimSpace(videoPart)
	if videoPart == "" {
		return "", "", fmt.Errorf("invalid video id in %q", path)
	}

	switch profilePart {
	case "", "retro":
		return ProfileRetro, videoPart, nil
	case "edge":
		return ProfileEdge, videoPart, nil
	default:
		return "", "", fmt.Errorf("unsupported profile %q", profilePart)
	}
}

func (r *rtspServer) getStream(path string) *rtspStream {
	key := canonicalPath(path)
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.streams[key]
}

func (r *rtspServer) getOrCreateStream(path string, profile Profile, videoID string) *rtspStream {
	key := canonicalPath(path)
	r.mu.Lock()
	defer r.mu.Unlock()
	if existing, ok := r.streams[key]; ok {
		return existing
	}
	stream := newRTSPStream(r, key, profile, videoID)
	r.streams[key] = stream
	return stream
}

func (r *rtspServer) registerPublisher(session *gortsplib.ServerSession, stream *rtspStream) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.publishers[session] = stream
}

func (r *rtspServer) streamByPublisher(session *gortsplib.ServerSession) *rtspStream {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.publishers[session]
}

func (r *rtspServer) removeStream(stream *rtspStream) {
	r.mu.Lock()
	if current, ok := r.streams[stream.path]; ok && current == stream {
		delete(r.streams, stream.path)
	}
	for sess, st := range r.publishers {
		if st == stream {
			delete(r.publishers, sess)
		}
	}
	r.mu.Unlock()

	stream.shutdown()
}

// OnDescribe handles DESCRIBE requests.
func (r *rtspServer) OnDescribe(ctx *gortsplib.ServerHandlerOnDescribeCtx) (*base.Response, *gortsplib.ServerStream, error) {
	profile, videoID, err := r.parsePath(ctx.Path)
	if err != nil {
		return &base.Response{StatusCode: base.StatusNotFound}, nil, nil
	}

	stream := r.getOrCreateStream(ctx.Path, profile, videoID)
	stream.ensureStarted()

	waitCtx, cancel := context.WithTimeout(context.Background(), rtspPublisherTimeout)
	defer cancel()

	srvStream, err := stream.waitReady(waitCtx)
	if err != nil {
		return &base.Response{StatusCode: base.StatusNotFound}, nil, nil
	}

	return &base.Response{StatusCode: base.StatusOK}, srvStream, nil
}

// OnAnnounce handles ANNOUNCE from the internal ffmpeg publisher.
func (r *rtspServer) OnAnnounce(ctx *gortsplib.ServerHandlerOnAnnounceCtx) (*base.Response, error) {
	stream := r.getStream(ctx.Path)
	if stream == nil {
		return &base.Response{StatusCode: base.StatusNotFound}, nil
	}

	if !isLocalPublisher(ctx.Conn.NetConn().RemoteAddr()) {
		return &base.Response{StatusCode: base.StatusForbidden}, nil
	}

	if err := stream.attachPublisher(ctx.Session, ctx.Description); err != nil {
		return &base.Response{StatusCode: base.StatusInternalServerError}, err
	}
	r.registerPublisher(ctx.Session, stream)
	return &base.Response{StatusCode: base.StatusOK}, nil
}

// OnSetup handles SETUP requests.
func (r *rtspServer) OnSetup(ctx *gortsplib.ServerHandlerOnSetupCtx) (*base.Response, *gortsplib.ServerStream, error) {
	// publisher path, no stream yet required.
	if ctx.Session.State() == gortsplib.ServerSessionStatePreRecord {
		return &base.Response{StatusCode: base.StatusOK}, nil, nil
	}

	stream := r.getStream(ctx.Path)
	if stream == nil {
		return &base.Response{StatusCode: base.StatusNotFound}, nil, nil
	}
	srvStream := stream.currentStream()
	if srvStream == nil {
		return &base.Response{StatusCode: base.StatusNotFound}, nil, nil
	}

	return &base.Response{StatusCode: base.StatusOK}, srvStream, nil
}

// OnPlay handles PLAY requests.
func (r *rtspServer) OnPlay(_ *gortsplib.ServerHandlerOnPlayCtx) (*base.Response, error) {
	return &base.Response{StatusCode: base.StatusOK}, nil
}

// OnRecord handles RECORD requests from the publisher.
func (r *rtspServer) OnRecord(ctx *gortsplib.ServerHandlerOnRecordCtx) (*base.Response, error) {
	stream := r.getStream(ctx.Path)
	if stream == nil {
		return &base.Response{StatusCode: base.StatusNotFound}, nil
	}
	stream.forwardPackets(ctx.Session)
	return &base.Response{StatusCode: base.StatusOK}, nil
}

// OnSessionClose cleans up streams when sessions terminate.
func (r *rtspServer) OnSessionClose(ctx *gortsplib.ServerHandlerOnSessionCloseCtx) {
	stream := r.streamByPublisher(ctx.Session)
	if stream != nil {
		r.removeStream(stream)
	}
}

func isLocalPublisher(addr net.Addr) bool {
	if addr == nil {
		return false
	}
	host, _, err := net.SplitHostPort(addr.String())
	if err != nil {
		return false
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	return ip.IsLoopback() || ip.IsUnspecified()
}

type rtspStream struct {
	server    *rtspServer
	path      string
	profile   Profile
	videoID   string
	startOnce sync.Once

	mu        sync.RWMutex
	stream    *gortsplib.ServerStream
	publisher *gortsplib.ServerSession
	cancel    context.CancelFunc
	cmd       *exec.Cmd
	err       error

	ready     chan struct{}
	readyOnce sync.Once
}

func newRTSPStream(server *rtspServer, path string, profile Profile, videoID string) *rtspStream {
	return &rtspStream{
		server:  server,
		path:    path,
		profile: profile,
		videoID: videoID,
		ready:   make(chan struct{}),
	}
}

func (rs *rtspStream) ensureStarted() {
	rs.startOnce.Do(func() {
		go rs.launchPublisher()
	})
}

func (rs *rtspStream) launchPublisher() {
	ctx, cancel := context.WithCancel(context.Background())
	rs.mu.Lock()
	rs.cancel = cancel
	rs.mu.Unlock()

	resolveCtx, resolveCancel := context.WithTimeout(ctx, rtspResolveTimeout)
	streamURL, err := rs.server.svc.resolveStream(resolveCtx, rs.videoID)
	resolveCancel()
	if err != nil {
		rs.fail(fmt.Errorf("resolve stream: %w", err))
		return
	}

	target := rs.publishURL()
	args, err := profileRTSPArgs(rs.profile, target)
	if err != nil {
		rs.fail(err)
		return
	}

	cmd := exec.CommandContext(ctx, rs.server.svc.command, args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		rs.fail(fmt.Errorf("stdin pipe: %w", err))
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		stdin.Close()
		rs.fail(fmt.Errorf("stderr pipe: %w", err))
		return
	}

	if err := cmd.Start(); err != nil {
		stdin.Close()
		rs.fail(fmt.Errorf("ffmpeg start: %w", err))
		return
	}

	rs.mu.Lock()
	rs.cmd = cmd
	rs.mu.Unlock()

	go rs.logStderr(stderr)
	go rs.pipeStream(ctx, stdin, streamURL)

	go func() {
		err := cmd.Wait()
		if err != nil && ctx.Err() == nil && !errors.Is(err, context.Canceled) {
			log.Printf("[rtsp] ffmpeg wait: %v", err)
		}
	}()
}

func (rs *rtspStream) publishURL() string {
	host := "127.0.0.1"
	port := rs.server.port
	if port == 0 {
		port = 8554
	}
	path := strings.TrimPrefix(rs.path, "/")
	return fmt.Sprintf("rtsp://%s:%d/%s", host, port, path)
}

func (rs *rtspStream) logStderr(r io.Reader) {
	scanner := newStderrScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.Contains(line, "frame=") {
			log.Printf("[ffmpeg rtsp] %s", line)
		}
	}
}

func (rs *rtspStream) pipeStream(ctx context.Context, stdin io.WriteCloser, srcURL string) {
	defer stdin.Close()
	for {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, srcURL, nil)
		if err != nil {
			rs.fail(fmt.Errorf("request build: %w", err))
			return
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")
		req.Header.Set("Referer", "https://www.youtube.com/")

		resp, err := rs.server.svc.client.Do(req)
		if err != nil {
			select {
			case <-ctx.Done():
				return
			default:
			}
			log.Printf("[rtsp fetch] %v", err)
			time.Sleep(rtspIngestRetryInterval)
			continue
		}

		_, err = io.Copy(stdin, resp.Body)
		resp.Body.Close()
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			rs.fail(fmt.Errorf("feed ffmpeg: %w", err))
		}
		return
	}
}

func (rs *rtspStream) waitReady(ctx context.Context) (*gortsplib.ServerStream, error) {
	select {
	case <-rs.ready:
		rs.mu.RLock()
		defer rs.mu.RUnlock()
		if rs.err != nil {
			return nil, rs.err
		}
		if rs.stream == nil {
			return nil, errors.New("rtsp: stream unavailable")
		}
		return rs.stream, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (rs *rtspStream) currentStream() *gortsplib.ServerStream {
	rs.mu.RLock()
	defer rs.mu.RUnlock()
	return rs.stream
}

func (rs *rtspStream) attachPublisher(session *gortsplib.ServerSession, desc *description.Session) error {
	stream := &gortsplib.ServerStream{
		Server: rs.server.server,
		Desc:   desc,
	}
	if err := stream.Initialize(); err != nil {
		rs.fail(fmt.Errorf("stream init: %w", err))
		return err
	}

	rs.mu.Lock()
	defer rs.mu.Unlock()
	if rs.publisher != nil && rs.publisher != session {
		stream.Close()
		return errors.New("rtsp: publisher already connected")
	}
	rs.publisher = session
	rs.stream = stream
	rs.readyOnce.Do(func() { close(rs.ready) })
	return nil
}

func (rs *rtspStream) forwardPackets(session *gortsplib.ServerSession) {
	session.OnPacketRTPAny(func(medi *description.Media, _ format.Format, pkt *rtp.Packet) {
		rs.mu.RLock()
		stream := rs.stream
		rs.mu.RUnlock()
		if stream != nil {
			if err := stream.WritePacketRTP(medi, pkt); err != nil {
				log.Printf("[rtsp] write packet: %v", err)
			}
		}
	})
}

func (rs *rtspStream) fail(err error) {
	rs.mu.Lock()
	if rs.err == nil {
		rs.err = err
	}
	cancel := rs.cancel
	rs.cancel = nil
	rs.mu.Unlock()

	if cancel != nil {
		cancel()
	}

	rs.readyOnce.Do(func() { close(rs.ready) })
	log.Printf("[rtsp] %v", err)
	rs.server.removeStream(rs)
}

func (rs *rtspStream) shutdown() {
	rs.mu.Lock()
	cancel := rs.cancel
	cmd := rs.cmd
	stream := rs.stream
	rs.cancel = nil
	rs.cmd = nil
	rs.stream = nil
	rs.publisher = nil
	rs.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if cmd != nil {
		_ = cmd.Process.Kill()
	}
	if stream != nil {
		stream.Close()
	}
}

func canonicalPath(path string) string {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return "/"
	}
	return "/" + trimmed
}
