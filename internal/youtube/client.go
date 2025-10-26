package youtube

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"youtube-mini/internal/platform/cache"
)

const (
	defaultSearchAgent = "Mozilla/5.0 (Mobile; rv:48.0) Gecko/48.0 Firefox/48.0 KAIOS/2.5.4"
	videoCacheTTL      = time.Hour
)

// Client wraps the handful of YouTube API calls we rely on.
type Client struct {
	apiKey       string
	httpClient   *http.Client
	searchAgent  string
	videoCache   *cache.Cache[Video]
	searchCache  *cache.Cache[[]SearchResult]
	homeCache    *cache.Cache[[]FeedItem]
	exploreCache *cache.Cache[[]FeedItem]
}

// New creates a client with sane defaults.
func New(apiKey string) *Client {
	return &Client{
		apiKey:       apiKey,
		httpClient:   &http.Client{Timeout: 15 * time.Second},
		searchAgent:  defaultSearchAgent,
		videoCache:   cache.New[Video](),
		searchCache:  cache.New[[]SearchResult](),
		homeCache:    cache.New[[]FeedItem](),
		exploreCache: cache.New[[]FeedItem](),
	}
}

// HTTPClient exposes the underlying HTTP client for ancillary fetches.
func (c *Client) HTTPClient() *http.Client {
	return c.httpClient
}

// Search calls the mobile YouTube search endpoint and returns simplified results.
func (c *Client) Search(ctx context.Context, query string) ([]SearchResult, error) {
	key := strings.ToLower(strings.TrimSpace(query))
	if res, ok := c.searchCache.Get(key); ok {
		return res, nil
	}
	body := map[string]any{
		"context": map[string]any{
			"client": map[string]any{
				"hl":            "ru",
				"gl":            "US",
				"clientName":    "MWEB",
				"clientVersion": "2.20251021.01.00",
			},
		},
		"query": query,
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("https://m.youtube.com/youtubei/v1/search?key=%s", c.apiKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", c.searchAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
		return nil, fmt.Errorf("search request failed: %s (%s)", resp.Status, string(bodyBytes))
	}

	var decoded map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, err
	}
	results := extractSearchResults(decoded)
	c.searchCache.Set(key, results, 5*time.Minute)
	return results, nil
}

// GetVideo returns playback metadata, pulling from cache when possible.
func (c *Client) GetVideo(ctx context.Context, id string) (Video, error) {
	if v, ok := c.videoCache.Get(id); ok {
		return v, nil
	}

	body := map[string]any{
		"videoId": id,
		"context": map[string]any{
			"client": map[string]any{
				"hl":            "en",
				"gl":            "US",
				"clientName":    "ANDROID",
				"clientVersion": "19.09.37",
			},
		},
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return Video{}, err
	}

	url := fmt.Sprintf("https://www.youtube.com/youtubei/v1/player?key=%s", c.apiKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return Video{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "com.google.android.youtube/19.09.37 (Linux; Android 2.3.7)")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return Video{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
		return Video{}, fmt.Errorf("player request failed: %s (%s)", resp.Status, string(bodyBytes))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return Video{}, err
	}

	var decoded struct {
		VideoDetails struct {
			Title         string `json:"title"`
			Author        string `json:"author"`
			LengthSeconds string `json:"lengthSeconds"`
			ViewCount     string `json:"viewCount"`
			ChannelID     string `json:"channelId"`
		} `json:"videoDetails"`
		StreamingData map[string]any `json:"streamingData"`
	}
	if err := json.Unmarshal(data, &decoded); err != nil {
		return Video{}, err
	}

	var generic map[string]any
	if err := json.Unmarshal(data, &generic); err != nil {
		generic = nil
	}

	videoFormats, audioFormats := parseFormats(decoded.StreamingData)
	captions := extractCaptions(generic)
	stream := ""
	if len(videoFormats) > 0 {
		stream = videoFormats[0].URL
	}

	video := Video{
		ID:            id,
		Title:         decoded.VideoDetails.Title,
		Author:        decoded.VideoDetails.Author,
		LengthSeconds: decoded.VideoDetails.LengthSeconds,
		ViewCount:     decoded.VideoDetails.ViewCount,
		ChannelID:     decoded.VideoDetails.ChannelID,
		Formats:       videoFormats,
		Audio:         audioFormats,
		Captions:      captions,
		Stream:        stream,
		ThumbURL:      fmt.Sprintf("https://i.ytimg.com/vi/%s/hqdefault.jpg", id),
	}
	c.videoCache.Set(id, video, videoCacheTTL)
	return video, nil
}

func extractSearchResults(obj map[string]any) []SearchResult {
	results := []SearchResult{}
	sections, _ := dig(obj, "contents", "sectionListRenderer", "contents").([]any)
	for _, section := range sections {
		items, _ := dig(section, "itemSectionRenderer", "contents").([]any)
		for _, item := range items {
			id, _ := dig(item, "videoWithContextRenderer", "videoId").(string)
			if id == "" {
				continue
			}

			title := fmt.Sprint(dig(item, "videoWithContextRenderer", "headline", "runs", 0, "text"))
			channel := fmt.Sprint(dig(item, "videoWithContextRenderer", "shortBylineText", "runs", 0, "text"))
			channelID := fmt.Sprint(dig(item, "videoWithContextRenderer", "shortBylineText", "runs", 0, "navigationEndpoint", "browseEndpoint", "browseId"))
			thumb := fmt.Sprint(dig(item, "videoWithContextRenderer", "thumbnail", "thumbnails", 0, "url"))

			duration := fmt.Sprint(dig(item, "videoWithContextRenderer", "lengthText", "simpleText"))
			if duration == "" || duration == "<nil>" {
				duration = fmt.Sprint(dig(item, "videoWithContextRenderer", "lengthText", "runs", 0, "text"))
			}

			views := fmt.Sprint(dig(item, "videoWithContextRenderer", "shortViewCountText", "simpleText"))
			if views == "" || views == "<nil>" {
				views = fmt.Sprint(dig(item, "videoWithContextRenderer", "shortViewCountText", "runs", 0, "text"))
			}

			published := fmt.Sprint(dig(item, "videoWithContextRenderer", "publishedTimeText", "simpleText"))
			if published == "" || published == "<nil>" {
				published = fmt.Sprint(dig(item, "videoWithContextRenderer", "publishedTimeText", "runs", 0, "text"))
			}

			metaParts := filterNonEmpty([]string{views, published})
			meta := strings.TrimSpace(strings.Join(metaParts, " - "))

			results = append(results, SearchResult{
				ID:        id,
				Title:     title,
				Channel:   channel,
				ChannelID: channelID,
				Thumbnail: thumb,
				Duration:  duration,
				Meta:      meta,
			})
		}
	}
	return results
}

func parseFormats(streaming map[string]any) (videoFormats []Format, audioFormats []Format) {
	rawFormats, _ := streaming["formats"].([]any)
	for _, raw := range rawFormats {
		if f := buildFormat(raw); f.URL != "" {
			videoFormats = append(videoFormats, f)
		}
	}

	rawAdaptive, _ := streaming["adaptiveFormats"].([]any)
	for _, raw := range rawAdaptive {
		f := buildFormat(raw)
		if f.URL == "" {
			continue
		}
		if strings.Contains(f.Mime, "audio") && !strings.Contains(f.Mime, "video") {
			audioFormats = append(audioFormats, f)
			continue
		}
		// Some adaptive entries are video-only.
		videoFormats = append(videoFormats, f)
	}
	return videoFormats, audioFormats
}

func buildFormat(raw any) Format {
	data, _ := raw.(map[string]any)
	if data == nil {
		return Format{}
	}
	return Format{
		Itag:    fmt.Sprint(data["itag"]),
		Mime:    fmt.Sprint(data["mimeType"]),
		URL:     fmt.Sprint(data["url"]),
		Quality: fmt.Sprint(data["qualityLabel"]),
		Bitrate: fmt.Sprint(data["bitrate"]),
	}
}

func extractCaptions(obj map[string]any) []CaptionTrack {
	tracks := []CaptionTrack{}
	if obj == nil {
		return tracks
	}
	rawTracks, _ := dig(obj, "captions", "playerCaptionsTracklistRenderer", "captionTracks").([]any)
	for _, raw := range rawTracks {
		m, _ := raw.(map[string]any)
		if m == nil {
			continue
		}
		url := fmt.Sprint(m["baseUrl"])
		if url == "" {
			continue
		}
		label := fmt.Sprint(dig(m, "name", "simpleText"))
		if label == "" || label == "<nil>" {
			label = fmt.Sprint(dig(m, "name", "runs", 0, "text"))
		}
		if label == "" || label == "<nil>" {
			label = fmt.Sprint(m["languageCode"])
		}
		kind := fmt.Sprint(m["kind"])
		tracks = append(tracks, CaptionTrack{
			Language: label,
			URL:      url,
			Kind:     kind,
		})
	}
	return tracks
}

// Home returns a cached set of recommended videos for unauthenticated users.
func (c *Client) Home(ctx context.Context) ([]FeedItem, error) {
	if items, ok := c.homeCache.Get("default"); ok {
		return items, nil
	}
	items, err := c.fetchBrowse(ctx, "FEwhat_to_watch")
	if err != nil {
		return nil, err
	}
	c.homeCache.Set("default", items, 5*time.Minute)
	return items, nil
}

// Trending returns a cached set of trending videos.
func (c *Client) Trending(ctx context.Context) ([]FeedItem, error) {
	if items, ok := c.exploreCache.Get("default"); ok {
		return items, nil
	}
	items, err := c.fetchBrowse(ctx, "FEtrending")
	if err != nil {
		return nil, err
	}
	c.exploreCache.Set("default", items, 5*time.Minute)
	return items, nil
}

func (c *Client) Channel(ctx context.Context, channelID string) (ChannelInfo, ChannelSections, error) {
	channelID = strings.TrimSpace(channelID)
	var emptyInfo ChannelInfo
	var emptySections ChannelSections
	if channelID == "" {
		return emptyInfo, emptySections, fmt.Errorf("channel id required")
	}

	body := map[string]any{
		"context": map[string]any{
			"client": map[string]any{
				"hl":            "en",
				"gl":            "US",
				"clientName":    "MWEB",
				"clientVersion": "2.20251021.01.00",
			},
		},
		"browseId": channelID,
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return emptyInfo, emptySections, err
	}
	url := fmt.Sprintf("https://www.youtube.com/youtubei/v1/browse?key=%s", c.apiKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return emptyInfo, emptySections, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", c.searchAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return emptyInfo, emptySections, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
		return emptyInfo, emptySections, fmt.Errorf("channel browse failed: %s (%s)", resp.Status, string(bodyBytes))
	}

	var decoded map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return emptyInfo, emptySections, err
	}

	header := dig(decoded, "header", "c4TabbedHeaderRenderer")
	info := ChannelInfo{
		ID:          channelID,
		Title:       cleanText(dig(header, "title")),
		AvatarURL:   cleanText(dig(header, "avatar", "thumbnails", 0, "url")),
		Subscribers: cleanText(dig(header, "subscriberCountText", "simpleText")),
		Description: cleanText(dig(decoded, "metadata", "channelMetadataRenderer", "description")),
	}

	tabs, _ := dig(decoded, "contents", "twoColumnBrowseResultsRenderer", "tabs").([]any)
	sections := ChannelSections{}
	for _, tabAny := range tabs {
		tab, _ := tabAny.(map[string]any)
		tabRenderer, _ := tab["tabRenderer"].(map[string]any)
		if tabRenderer == nil {
			continue
		}
		title := strings.ToLower(cleanText(tabRenderer["title"]))
		content := tabRenderer["content"]
		var renderers []map[string]any
		collectVideoRenderers(content, &renderers)
		items := convertRenderers(renderers)
		if len(items) == 0 {
			continue
		}
		switch {
		case strings.Contains(title, "popular"):
			if len(sections.Popular) == 0 {
				sections.Popular = items
			}
		case strings.Contains(title, "short"):
			if len(sections.Shorts) == 0 {
				sections.Shorts = items
			}
		case strings.Contains(title, "live"):
			if len(sections.Live) == 0 {
				sections.Live = items
			}
		default:
			if len(sections.Latest) == 0 {
				sections.Latest = items
			}
		}
	}

	return info, sections, nil
}

func convertRenderers(renderers []map[string]any) []FeedItem {
	items := make([]FeedItem, 0, len(renderers))
	seen := make(map[string]struct{})
	for _, renderer := range renderers {
		item, ok := feedItemFromRenderer(renderer)
		if !ok || item.ID == "" {
			continue
		}
		if _, exists := seen[item.ID]; exists {
			continue
		}
		seen[item.ID] = struct{}{}
		items = append(items, item)
		if len(items) >= 40 {
			break
		}
	}
	return items
}

// ChannelFeed returns recent uploads for a specific channel.
func (c *Client) ChannelFeed(ctx context.Context, channelID string) ([]FeedItem, error) {
	channelID = strings.TrimSpace(channelID)
	if channelID == "" {
		return nil, nil
	}
	return c.fetchBrowse(ctx, channelID)
}

// Next fetches related videos and the autoplay candidate for the given id.
func (c *Client) Next(ctx context.Context, id string) ([]FeedItem, string, error) {
	body := map[string]any{
		"context": map[string]any{
			"client": map[string]any{
				"hl":            "en",
				"gl":            "US",
				"clientName":    "MWEB",
				"clientVersion": "2.20251021.01.00",
			},
		},
		"videoId": id,
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, "", err
	}
	url := fmt.Sprintf("https://www.youtube.com/youtubei/v1/next?key=%s", c.apiKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", c.searchAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
		return nil, "", fmt.Errorf("next request failed: %s (%s)", resp.Status, string(bodyBytes))
	}

	var decoded map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, "", err
	}

	var renderers []map[string]any
	results := dig(decoded, "contents", "twoColumnWatchNextResults", "secondaryResults", "secondaryResults", "results")
	collectVideoRenderers(results, &renderers)

	items := make([]FeedItem, 0, len(renderers))
	for _, renderer := range renderers {
		if item, ok := feedItemFromRenderer(renderer); ok {
			items = append(items, item)
		}
	}

	nextID := fmt.Sprint(dig(decoded, "contents", "twoColumnWatchNextResults", "autoplay", "autoplayRenderer", "contents", 0, "autoplayVideoRenderer", "watchEndpoint", "videoId"))
	return items, nextID, nil
}

// Suggest returns lightweight search suggestions for the query.
func (c *Client) Suggest(ctx context.Context, query string) ([]string, error) {
	if strings.TrimSpace(query) == "" {
		return nil, nil
	}
	endpoint := fmt.Sprintf("https://suggestqueries.google.com/complete/search?client=firefox&ds=yt&q=%s", url.QueryEscape(query))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
		return nil, fmt.Errorf("suggest request failed: %s (%s)", resp.Status, string(bodyBytes))
	}

	var payload []any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	if len(payload) < 2 {
		return nil, nil
	}
	rawSuggestions, _ := payload[1].([]any)
	suggestions := make([]string, 0, len(rawSuggestions))
	for _, entry := range rawSuggestions {
		suggestions = append(suggestions, fmt.Sprint(entry))
	}
	return suggestions, nil
}

func (c *Client) fetchBrowse(ctx context.Context, browseID string) ([]FeedItem, error) {
	body := map[string]any{
		"context": map[string]any{
			"client": map[string]any{
				"hl":            "en",
				"gl":            "US",
				"clientName":    "MWEB",
				"clientVersion": "2.20251021.01.00",
			},
		},
		"browseId": browseID,
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("https://www.youtube.com/youtubei/v1/browse?key=%s", c.apiKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", c.searchAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
		return nil, fmt.Errorf("browse request failed: %s (%s)", resp.Status, string(bodyBytes))
	}

	var decoded map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, err
	}

	var renderers []map[string]any
	collectVideoRenderers(decoded, &renderers)

	items := make([]FeedItem, 0, len(renderers))
	seen := make(map[string]struct{})
	for _, renderer := range renderers {
		item, ok := feedItemFromRenderer(renderer)
		if !ok || item.ID == "" {
			continue
		}
		if _, exists := seen[item.ID]; exists {
			continue
		}
		seen[item.ID] = struct{}{}
		items = append(items, item)
		if len(items) >= 40 {
			break
		}
	}
	return items, nil
}

func collectVideoRenderers(node any, out *[]map[string]any) {
	switch typed := node.(type) {
	case map[string]any:
		if vr, ok := typed["videoRenderer"].(map[string]any); ok {
			*out = append(*out, vr)
		}
		if cvr, ok := typed["compactVideoRenderer"].(map[string]any); ok {
			*out = append(*out, cvr)
		}
		if pvr, ok := typed["playlistVideoRenderer"].(map[string]any); ok {
			*out = append(*out, pvr)
		}
		if gvr, ok := typed["gridVideoRenderer"].(map[string]any); ok {
			*out = append(*out, gvr)
		}
		for _, v := range typed {
			collectVideoRenderers(v, out)
		}
	case []any:
		for _, v := range typed {
			collectVideoRenderers(v, out)
		}
	}
}

func feedItemFromRenderer(m map[string]any) (FeedItem, bool) {
	id := cleanText(m["videoId"])
	if id == "" {
		return FeedItem{}, false
	}

	title := cleanText(dig(m, "title", "simpleText"))
	if title == "" {
		title = cleanText(dig(m, "title", "runs", 0, "text"))
	}

	channel := cleanText(dig(m, "longBylineText", "runs", 0, "text"))
	if channel == "" {
		channel = cleanText(dig(m, "shortBylineText", "runs", 0, "text"))
	}
	if channel == "" {
		channel = cleanText(dig(m, "ownerText", "runs", 0, "text"))
	}
	if channel == "" {
		channel = cleanText(dig(m, "videoOwnerRenderer", "title", "runs", 0, "text"))
	}
	if channel == "" {
		channel = cleanText(dig(m, "channelThumbnailSupportedRenderers", "channelThumbnailWithLinkRenderer", "tooltip"))
	}

	channelID := cleanText(dig(m, "longBylineText", "runs", 0, "navigationEndpoint", "browseEndpoint", "browseId"))
	if channelID == "" {
		channelID = cleanText(dig(m, "shortBylineText", "runs", 0, "navigationEndpoint", "browseEndpoint", "browseId"))
	}
	if channelID == "" {
		channelID = cleanText(dig(m, "ownerText", "runs", 0, "navigationEndpoint", "browseEndpoint", "browseId"))
	}
	if channelID == "" {
		channelID = cleanText(dig(m, "videoOwnerRenderer", "navigationEndpoint", "browseEndpoint", "browseId"))
	}
	if channelID == "" {
		channelID = cleanText(dig(m, "channelThumbnailSupportedRenderers", "channelThumbnailWithLinkRenderer", "navigationEndpoint", "browseEndpoint", "browseId"))
	}
	if channelID == "" {
		channelID = cleanText(dig(m, "navigationEndpoint", "browseEndpoint", "browseId"))
	}

	thumb := cleanText(dig(m, "thumbnail", "thumbnails", 0, "url"))

	duration := cleanText(dig(m, "lengthText", "simpleText"))
	if duration == "" {
		duration = cleanText(dig(m, "lengthText", "runs", 0, "text"))
	}
	if duration == "" {
		duration = cleanText(dig(m, "lengthSeconds"))
	}

	views := cleanText(dig(m, "viewCountText", "simpleText"))
	if views == "" {
		views = cleanText(dig(m, "shortViewCountText", "simpleText"))
	}
	if views == "" {
		views = cleanText(dig(m, "shortViewCountText", "runs", 0, "text"))
	}

	published := cleanText(dig(m, "publishedTimeText", "simpleText"))
	if published == "" {
		published = cleanText(dig(m, "publishedTimeText", "runs", 0, "text"))
	}

	meta := strings.TrimSpace(strings.Join(filterNonEmpty([]string{views, published}), " - "))

	return FeedItem{
		ID:        id,
		Title:     title,
		Channel:   channel,
		ChannelID: channelID,
		Thumbnail: thumb,
		Duration:  duration,
		Meta:      meta,
	}, true
}

func filterNonEmpty(items []string) []string {
	out := make([]string, 0, len(items))
	for _, it := range items {
		if it != "" && it != "<nil>" {
			out = append(out, it)
		}
	}
	return out
}

func dig(v any, keys ...any) any {
	cur := v
	for _, k := range keys {
		switch key := k.(type) {
		case string:
			if m, ok := cur.(map[string]any); ok {
				cur = m[key]
			} else {
				return nil
			}
		case int:
			if a, ok := cur.([]any); ok {
				if key >= 0 && key < len(a) {
					cur = a[key]
				} else {
					return nil
				}
			} else {
				return nil
			}
		}
	}
	return cur
}

func cleanText(v any) string {
	s := strings.TrimSpace(fmt.Sprint(v))
	if s == "<nil>" {
		return ""
	}
	return s
}
