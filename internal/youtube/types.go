package youtube

// SearchResult represents a single search card returned by the mobile API.
type SearchResult struct {
	ID        string
	Title     string
	Channel   string
	ChannelID string
	Thumbnail string
	Duration  string
	Meta      string
}

// Format is a video stream rendition.
type Format struct {
	Itag    string
	Mime    string
	URL     string
	Quality string
	Bitrate string
}

// CaptionTrack describes an available subtitle track.
type CaptionTrack struct {
	Language string
	URL      string
	Kind     string
}

// Video is the aggregate metadata needed for the watch page.
type Video struct {
	ID            string
	Title         string
	Author        string
	LengthSeconds string
	ViewCount     string
	ChannelID     string
	Formats       []Format
	Audio         []Format
	Captions      []CaptionTrack
	Stream        string
	ThumbURL      string
}

// FeedItem represents a lightweight video card for feeds like home or trending.
type FeedItem struct {
	ID        string
	Title     string
	Channel   string
	ChannelID string
	Thumbnail string
	Duration  string
	Meta      string
}

// ChannelInfo holds metadata about a channel.
type ChannelInfo struct {
	ID          string
	Title       string
	AvatarURL   string
	Subscribers string
	Description string
}

// ChannelSections groups channel content buckets.
type ChannelSections struct {
	Latest  []FeedItem
	Popular []FeedItem
	Shorts  []FeedItem
	Live    []FeedItem
}
