package transcode

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"net/url"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bluenviron/gortsplib/v4"
	"github.com/bluenviron/gortsplib/v4/pkg/base"
	"github.com/bluenviron/gortsplib/v4/pkg/description"
	"github.com/bluenviron/gortsplib/v4/pkg/format"
	"github.com/bluenviron/gortsplib/v4/pkg/headers"
	"github.com/pion/rtp"
)

const (
	defaultRTSPAddress   = ":8554"
	rtspPublisherTimeout = 12 * time.Second
	rtspResolveTimeout   = 15 * time.Second
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
		Handler:        rs,
		RTSPAddress:    addr,
		UDPRTPAddress:  svc.udpRTPAddr,
		UDPRTCPAddress: svc.udpRTCPAddr,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
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
	case ProfileAndroid:
		return "android"
	case ProfileRetro, "":
		return "retro"
	default:
		return string(profile)
	}
}

func (r *rtspServer) parsePath(path, query string) (Profile, string, float64, string, error) {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return "", "", 0, "", errors.New("empty path")
	}
	parts := strings.Split(trimmed, "/")
	if len(parts) < 2 {
		return "", "", 0, "", fmt.Errorf("invalid path %q", path)
	}
	profilePart := strings.ToLower(parts[0])
	videoPart := strings.Join(parts[1:], "/")
	videoPart = strings.TrimSuffix(videoPart, ".3gp")
	videoPart = strings.TrimSpace(videoPart)
	if videoPart == "" {
		return "", "", 0, "", fmt.Errorf("invalid video id in %q", path)
	}

	start := parseStartFromQuery(query)
	transport := transportFromQuery(query, r.svc.rtspTransport)

	switch profilePart {
	case "", "retro":
		return ProfileRetro, videoPart, start, transport, nil
	case "edge":
		return ProfileEdge, videoPart, start, transport, nil
	case "android":
		return ProfileAndroid, videoPart, start, transport, nil
	default:
		return "", "", 0, "", fmt.Errorf("unsupported profile %q", profilePart)
	}
}

func (r *rtspServer) getStream(path, query string) *rtspStream {
	key := canonicalKey(path, query)
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.streams[key]
}

func (r *rtspServer) getOrCreateStream(path, query string, profile Profile, videoID string, start float64, transport string) *rtspStream {
	key := canonicalKey(path, query)
	r.mu.Lock()
	defer r.mu.Unlock()
	if existing, ok := r.streams[key]; ok {
		return existing
	}
	stream := newRTSPStream(r, key, canonicalPath(path), canonicalQuery(query), profile, videoID, start, transport)
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
	if current, ok := r.streams[stream.key]; ok && current == stream {
		delete(r.streams, stream.key)
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
	profile, videoID, start, transport, err := r.parsePath(ctx.Path, ctx.Query)
	if err != nil {
		return &base.Response{StatusCode: base.StatusNotFound}, nil, nil
	}

	stream := r.getOrCreateStream(ctx.Path, ctx.Query, profile, videoID, start, transport)
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
	stream := r.getStream(ctx.Path, ctx.Query)
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

	stream := r.getStream(ctx.Path, ctx.Query)
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
func (r *rtspServer) OnPlay(ctx *gortsplib.ServerHandlerOnPlayCtx) (*base.Response, error) {
	stream := r.getStream(ctx.Path, ctx.Query)
	if stream == nil {
		return &base.Response{StatusCode: base.StatusNotFound}, nil
	}

	if offset, ok := rangeStartSeconds(ctx.Request); ok {
		if err := stream.seek(offset); err != nil {
			return &base.Response{StatusCode: base.StatusInvalidRange}, err
		}
	}

	return &base.Response{StatusCode: base.StatusOK}, nil
}

// OnRecord handles RECORD requests from the publisher.
func (r *rtspServer) OnRecord(ctx *gortsplib.ServerHandlerOnRecordCtx) (*base.Response, error) {
	stream := r.getStream(ctx.Path, ctx.Query)
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
	server      *rtspServer
	key         string
	path        string
	query       string
	profile     Profile
	videoID     string
	startOffset float64
	transport   string

	mu        sync.RWMutex
	stream    *gortsplib.ServerStream
	publisher *gortsplib.ServerSession
	cancel    context.CancelFunc
	cmd       *exec.Cmd
	err       error
	cleanup   func()
	running   bool
	ready     chan struct{}
}

func newRTSPStream(server *rtspServer, key, path, query string, profile Profile, videoID string, start float64, transport string) *rtspStream {
	return &rtspStream{
		server:      server,
		key:         key,
		path:        path,
		query:       query,
		profile:     profile,
		videoID:     videoID,
		startOffset: start,
		transport:   transport,
		ready:       make(chan struct{}),
	}
}

func (rs *rtspStream) ensureStarted() {
	rs.mu.Lock()
	if rs.running {
		rs.mu.Unlock()
		return
	}
	if rs.ready == nil {
		rs.ready = make(chan struct{})
	}
	rs.running = true
	rs.mu.Unlock()

	go rs.launchPublisher()
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
		rs.setRunning(false)
		rs.fail(fmt.Errorf("resolve stream: %w", err))
		return
	}

	input, cleanup, err := rs.server.svc.buildInput(streamURL, rs.startOffset)
	if err != nil {
		rs.setRunning(false)
		rs.fail(err)
		return
	}
	rs.setCleanup(cleanup)

	target := rs.publishURL()
	transport := rs.transport
	args, err := rs.server.svc.profileRTSPArgs(rs.profile, input, target, transport)
	if err != nil {
		if cb := rs.takeCleanup(); cb != nil {
			cb()
		}
		rs.setRunning(false)
		rs.fail(err)
		return
	}

	cmd := exec.CommandContext(ctx, rs.server.svc.command, args...)
	var stdin io.WriteCloser
	if input.pipe {
		stdInPipe, err := cmd.StdinPipe()
		if err != nil {
			if cb := rs.takeCleanup(); cb != nil {
				cb()
			}
			rs.setRunning(false)
			rs.fail(fmt.Errorf("stdin pipe: %w", err))
			return
		}
		stdin = stdInPipe
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		if stdin != nil {
			_ = stdin.Close()
		}
		if cb := rs.takeCleanup(); cb != nil {
			cb()
		}
		rs.setRunning(false)
		rs.fail(fmt.Errorf("stderr pipe: %w", err))
		return
	}

	if err := cmd.Start(); err != nil {
		if stdin != nil {
			_ = stdin.Close()
		}
		if cb := rs.takeCleanup(); cb != nil {
			cb()
		}
		rs.setRunning(false)
		rs.fail(fmt.Errorf("ffmpeg start: %w", err))
		return
	}

	if input.pipe && stdin != nil {
		rs.server.svc.startInputPump(ctx, stdin, input.srcURL)
	}

	rs.mu.Lock()
	rs.cmd = cmd
	rs.mu.Unlock()

	go rs.logStderr(stderr)

	go func() {
		err := cmd.Wait()
		if err != nil && ctx.Err() == nil && !errors.Is(err, context.Canceled) {
			log.Printf("[rtsp] ffmpeg wait: %v", err)
		}
		if cb := rs.takeCleanup(); cb != nil {
			cb()
		}
		rs.mu.Lock()
		if rs.cmd == cmd {
			rs.cmd = nil
		}
		rs.running = false
		rs.mu.Unlock()
	}()
}

func (rs *rtspStream) publishURL() string {
	host := "127.0.0.1"
	port := rs.server.port
	if port == 0 {
		port = 8554
	}
	path := strings.TrimPrefix(rs.path, "/")
	if rs.query != "" {
		path = fmt.Sprintf("%s?%s", path, rs.query)
	}
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

func (rs *rtspStream) seek(offset float64) error {
	if offset < 0 {
		offset = 0
	}

	rs.mu.Lock()
	current := rs.startOffset
	oldReady := rs.ready
	oldCancel := rs.cancel
	oldCleanup := rs.cleanup
	if math.Abs(current-offset) < 0.5 && rs.stream != nil {
		rs.mu.Unlock()
		return nil
	}
	rs.startOffset = offset
	rs.cancel = nil
	rs.cmd = nil
	rs.err = nil
	rs.cleanup = nil
	rs.running = false
	rs.ready = make(chan struct{})
	rs.publisher = nil
	rs.mu.Unlock()

	if oldCancel != nil {
		oldCancel()
	}
	if oldCleanup != nil {
		oldCleanup()
	}
	safeClose(oldReady)

	rs.ensureStarted()

	waitCtx, cancel := context.WithTimeout(context.Background(), rtspPublisherTimeout)
	defer cancel()
	_, err := rs.waitReady(waitCtx)
	return err
}

func (rs *rtspStream) setRunning(state bool) {
	rs.mu.Lock()
	rs.running = state
	rs.mu.Unlock()
}

func (rs *rtspStream) setCleanup(fn func()) {
	rs.mu.Lock()
	rs.cleanup = fn
	rs.mu.Unlock()
}

func (rs *rtspStream) takeCleanup() func() {
	rs.mu.Lock()
	fn := rs.cleanup
	rs.cleanup = nil
	rs.mu.Unlock()
	return fn
}

func (rs *rtspStream) waitReady(ctx context.Context) (*gortsplib.ServerStream, error) {
	for {
		rs.mu.RLock()
		ready := rs.ready
		stream := rs.stream
		publisher := rs.publisher
		err := rs.err
		rs.mu.RUnlock()

		if err != nil {
			return nil, err
		}
		if publisher != nil && stream != nil {
			return stream, nil
		}
		if ready == nil {
			ready = make(chan struct{})
			rs.mu.Lock()
			if rs.ready == nil {
				rs.ready = ready
			} else {
				ready = rs.ready
			}
			rs.mu.Unlock()
		}

		select {
		case <-ready:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

func (rs *rtspStream) currentStream() *gortsplib.ServerStream {
	rs.mu.RLock()
	defer rs.mu.RUnlock()
	return rs.stream
}

func (rs *rtspStream) attachPublisher(session *gortsplib.ServerSession, desc *description.Session) error {
	rs.mu.Lock()
	if rs.publisher != nil && rs.publisher != session {
		rs.mu.Unlock()
		return errors.New("rtsp: publisher already connected")
	}
	stream := rs.stream
	if stream == nil {
		stream = &gortsplib.ServerStream{
			Server: rs.server.server,
			Desc:   desc,
		}
		if err := stream.Initialize(); err != nil {
			rs.mu.Unlock()
			rs.fail(fmt.Errorf("stream init: %w", err))
			return err
		}
		rs.stream = stream
	} else {
		stream.Desc = desc
	}
	rs.publisher = session
	ready := rs.ready
	rs.mu.Unlock()

	safeClose(ready)
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
	cleanup := rs.cleanup
	ready := rs.ready
	rs.cancel = nil
	rs.cleanup = nil
	rs.running = false
	rs.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if cleanup != nil {
		cleanup()
	}

	safeClose(ready)
	log.Printf("[rtsp] %v", err)
	rs.server.removeStream(rs)
}

func (rs *rtspStream) shutdown() {
	rs.mu.Lock()
	cancel := rs.cancel
	cmd := rs.cmd
	stream := rs.stream
	cleanup := rs.cleanup
	ready := rs.ready
	rs.cancel = nil
	rs.cmd = nil
	rs.stream = nil
	rs.publisher = nil
	rs.cleanup = nil
	rs.running = false
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
	if cleanup != nil {
		cleanup()
	}
	safeClose(ready)
}

func safeClose(ch chan struct{}) {
	if ch == nil {
		return
	}
	defer func() {
		_ = recover()
	}()
	close(ch)
}

func canonicalPath(path string) string {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return "/"
	}
	return "/" + trimmed
}

func canonicalQuery(query string) string {
	if query == "" {
		return ""
	}
	values, err := url.ParseQuery(query)
	if err != nil {
		return query
	}
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	canonical := url.Values{}
	for _, k := range keys {
		vals := values[k]
		sort.Strings(vals)
		for _, v := range vals {
			canonical.Add(k, v)
		}
	}
	return canonical.Encode()
}

func canonicalKey(path, query string) string {
	base := canonicalPath(path)
	if query == "" {
		return base
	}
	cq := canonicalQuery(query)
	if cq == "" {
		return base
	}
	return base + "?" + cq
}

func parseStartFromQuery(rawQuery string) float64 {
	if rawQuery == "" {
		return 0
	}
	values, err := url.ParseQuery(rawQuery)
	if err != nil {
		return 0
	}
	spec := strings.TrimSpace(values.Get("start"))
	if spec == "" {
		spec = strings.TrimSpace(values.Get("t"))
	}
	if secs, ok := ParseTimeSpec(spec); ok {
		return secs
	}
	return 0
}

func transportFromQuery(rawQuery, fallback string) string {
	if rawQuery == "" {
		return fallback
	}
	values, err := url.ParseQuery(rawQuery)
	if err != nil {
		return fallback
	}
	for _, key := range []string{"transport", "rtsp_transport"} {
		if v := strings.ToLower(strings.TrimSpace(values.Get(key))); v != "" {
			switch v {
			case "tcp", "udp", "udp_multicast":
				return v
			case "auto":
				return ""
			default:
				log.Printf("[rtsp] unsupported transport query %q, using fallback", v)
			}
		}
	}
	return fallback
}

func rangeStartSeconds(req *base.Request) (float64, bool) {
	if req == nil {
		return 0, false
	}
	raw, ok := req.Header["Range"]
	if !ok || len(raw) == 0 {
		return 0, false
	}
	var h headers.Range
	if err := h.Unmarshal(raw); err != nil {
		return 0, false
	}
	npt, ok := h.Value.(*headers.RangeNPT)
	if !ok {
		return 0, false
	}
	return npt.Start.Seconds(), true
}
