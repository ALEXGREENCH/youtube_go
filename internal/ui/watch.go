package ui

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"youtube-mini/internal/youtube"
)

// WatchPageData aggregates everything required to render the watch view.
type WatchPageData struct {
	Theme             string
	CurrentPath       string
	Video             youtube.Video
	StreamURL         string
	AudioURL          string
	TranscodeLinks    []Link
	Captions          []youtube.CaptionTrack
	AutoplayEnabled   bool
	AutoplayToggleURL string
	AutoplayNextURL   string
	AutoplaySource    string
	Related           []RelatedEntry
	Queue             []QueueEntry
	QueueClearURL     string
	Subscribed        bool
	SubscribeURL      string
	UnsubscribeURL    string
	InWatchLater      bool
	WatchLaterURL     string
}

// Link represents a simple label/url pair.
type Link struct {
	Label string
	URL   string
}

// RelatedEntry wraps a feed item with queue action.
type RelatedEntry struct {
	Item          youtube.FeedItem
	AddToQueueURL string
}

// QueueEntry contains queue metadata and removal action.
type QueueEntry struct {
	Item      youtube.FeedItem
	RemoveURL string
}

// RenderWatch renders the watch page with related videos, queue, and controls.
func RenderWatch(data WatchPageData) string {
	video := data.Video
	progressiveFormats := filterProgressiveFormats(video.Formats)

	primaryQuality, primarySize := primaryFormatMeta(progressiveFormats)
	if primaryQuality == "" && primarySize == "" {
		primaryQuality, primarySize = primaryFormatMeta(video.Formats)
	}
	primaryMeta := buildMetaBadge(primaryQuality, primarySize)

	mp4Download := data.StreamURL
	if len(progressiveFormats) > 0 && progressiveFormats[0].URL != "" {
		mp4Download = progressiveFormats[0].URL
	}

	liteLink, hasLite := findLinkByKeyword(data.TranscodeLinks, "3gp", "retro", "edge")
	liteURL := mp4Download
	liteLabel := "MP4 stream"
	if hasLite && liteLink.URL != "" {
		liteURL = liteLink.URL
		liteLabel = liteLink.Label
	}
	liteActions := buildLiteActionLinks(Link{Label: liteLabel, URL: liteURL}, hasLite && liteLink.URL != "")

	playerStream := data.StreamURL
	if mp4Download != "" {
		playerStream = mp4Download
	}

	previewAttr := EscapeAttr(ProxiedImage(video.ThumbURL))
	buttonMeta := primaryMeta
	if strings.TrimSpace(buttonMeta) == "" {
		buttonMeta = "MP4 stream"
	}

	var b strings.Builder

	b.WriteString(`<!DOCTYPE html><html><head><meta charset="utf-8">`)
	fmt.Fprintf(&b, `<title>%s - YouTube Mini</title>
<link rel="stylesheet" href="/style.css">
`, Escape(video.Title))

	b.WriteString(`<meta name="viewport" content="width=device-width, initial-scale=1.0">
<style>
:root{
 --ym-bg: rgba(18, 18, 18, 0.9);
 --ym-fg: #fff;
 --ym-muted: rgba(255, 255, 255, 0.65);
 --ym-divider: rgba(255, 255, 255, 0.12);
 --ym-card: rgba(255, 255, 255, 0.08);
 --ym-button-bg: #ff0000;
 --ym-button-fg: #fff;
 --ym-button-muted: rgba(255, 255, 255, 0.12);
 --ym-shadow: 0 8px 24px rgba(0, 0, 0, 0.35);
}
.light :root,
.theme-light :root,
.theme-light{
 --ym-bg: rgba(255, 255, 255, 0.94);
 --ym-fg: #111;
 --ym-muted: rgba(17, 17, 17, 0.65);
 --ym-divider: rgba(17, 17, 17, 0.08);
 --ym-card: rgba(17, 17, 17, 0.03);
 --ym-button-bg: #ff0000;
 --ym-button-fg: #fff;
 --ym-button-muted: rgba(17, 17, 17, 0.1);
 --ym-shadow: 0 12px 32px rgba(0, 0, 0, 0.12);
}
.watch-hero{
 background: var(--ym-card);
 border-radius: 18px;
 box-shadow: var(--ym-shadow);
 padding: 0;
 overflow:hidden;
}
.ym-player{
 max-width: 360px;
 margin: 0 auto;
 display:flex;
 flex-direction:column;
 align-items:center;
 background: var(--ym-bg);
 color: var(--ym-fg);
 border-bottom:1px solid var(--ym-divider);
}
.ym-preview{
 width:100%;
 padding:16px;
 text-align:center;
 background:linear-gradient(135deg, rgba(255,0,0,0.12), transparent 55%);
}
.ym-preview-img{
 width:100%;
 border-radius:16px;
 box-shadow:0 10px 28px rgba(0, 0, 0, 0.4);
}
.ym-play-button{
 display:inline-flex;
 align-items:center;
 justify-content:center;
 gap:10px;
 width:100%;
 margin-top:14px;
 padding:12px 18px;
 font-size:18px;
 font-weight:700;
 background:var(--ym-button-bg);
 color:var(--ym-button-fg);
 border:0;
 border-radius:999px;
 cursor:pointer;
 transition:transform 0.14s ease, box-shadow 0.14s ease;
 box-shadow:0 6px 18px rgba(255,0,0,0.35);
}
.ym-play-button::before{
 content:'▶';
 font-size:16px;
}
.ym-play-button:hover{
 transform:translateY(-1px);
 box-shadow:0 12px 28px rgba(255,0,0,0.45);
}
.ym-play-button:active{
 transform:translateY(1px);
}
.ym-play-button .ym-meta{
 font-size:12px;
 font-weight:400;
 opacity:0.85;
}
.ym-support-hint{
 margin-top:10px;
 font-size:13px;
 opacity:0.75;
}
.ym-video{
 width:100%;
 max-height:220px;
 background-color:#000;
 border-radius:18px;
}
.ym-lite-panel{
 padding:16px;
 text-align:center;
 background:rgba(0,0,0,0.15);
 color:var(--ym-fg);
}
.light .ym-lite-panel{
 background:rgba(17,17,17,0.06);
}
.ym-lite-panel img{
 width:100%;
 border-radius:14px;
 margin-bottom:12px;
 box-shadow: inset 0 0 0 1px rgba(255,255,255,0.05);
}
.ym-button{
 display:inline-block;
 margin:4px 3px;
 padding:9px 18px;
 border-radius:999px;
 font-size:14px;
 font-weight:600;
 text-decoration:none;
 border:1px solid transparent;
 transition:transform 0.12s ease, box-shadow 0.12s ease;
}
.ym-button-primary{
 background:var(--ym-button-bg);
 color:var(--ym-button-fg);
 box-shadow:0 6px 18px rgba(255,0,0,0.32);
}
.ym-button-ghost{
 background:var(--ym-button-muted);
 color:var(--ym-fg);
 border-color:rgba(255,255,255,0.1);
}
.light .ym-button-ghost{
 border-color:rgba(17,17,17,0.12);
}
.ym-button:hover{
 transform:translateY(-1px);
}
.ym-chip{
 display:inline-block;
 margin:4px 6px 0 0;
 padding:6px 12px;
 border-radius:999px;
 font-size:12px;
 border:1px solid var(--ym-divider);
 color:var(--ym-muted);
}
.ym-meta-panel{
 padding:18px;
 font-size:14px;
 color:var(--ym-fg);
 background:var(--ym-bg);
}
.ym-title{
 margin:0 0 14px;
 font-size:19px;
 font-weight:700;
 color:var(--ym-fg);
 line-height:1.35;
}
.ym-meta-panel b{
 display:block;
 font-size:17px;
 margin-bottom:6px;
}
.ym-meta-panel small a{
 color:var(--ym-button-bg);
 text-decoration:none;
}
.ym-meta-panel small{
 display:block;
 margin:3px 0;
 color:var(--ym-muted);
}
.ym-meta-header{
 display:flex;
 align-items:flex-start;
 justify-content:space-between;
 gap:14px;
 margin-bottom:12px;
}
.ym-channel-block{
 display:flex;
 align-items:center;
 gap:12px;
 flex:1 1 auto;
}
.ym-avatar{
 width:44px;
 height:44px;
 border-radius:50%;
 background:linear-gradient(135deg,#ff0000,rgba(255,0,0,0.35));
 color:#fff;
 display:flex;
 align-items:center;
 justify-content:center;
 font-weight:700;
 font-size:18px;
 text-transform:uppercase;
 box-shadow:0 6px 18px rgba(255,0,0,0.35);
}
.ym-channel-details{
 display:flex;
 flex-direction:column;
 gap:2px;
}
.ym-channel-details a,
.ym-channel-details span{
 color:var(--ym-fg);
 text-decoration:none;
}
.ym-channel-details a:hover{
 text-decoration:underline;
}
.ym-channel-count{
 font-size:12px;
 color:var(--ym-muted);
}
.ym-meta-stats{
 display:flex;
 flex-wrap:wrap;
 gap:10px;
 font-size:13px;
 color:var(--ym-muted);
 margin-bottom:8px;
}
.ym-meta-stats span{
 display:inline-flex;
 align-items:center;
 gap:4px;
}
.ym-meta-stats span::before{
 content:'•';
 opacity:0.45;
 margin-right:6px;
}
.ym-meta-stats span:first-child::before{
 content:'';
 margin-right:0;
 display:none;
}
.ym-sub-btn{
 display:inline-flex;
 align-items:center;
 justify-content:center;
 padding:8px 18px;
 border-radius:999px;
 font-size:14px;
 font-weight:600;
 border:1px solid var(--ym-button-bg);
 background:transparent;
 color:var(--ym-button-bg);
 cursor:pointer;
 transition:transform 0.12s ease, background 0.12s ease;
}
.ym-sub-btn:hover{
 transform:translateY(-1px);
 background:rgba(255,0,0,0.1);
}
.ym-sub-btn.is-on{
 background:var(--ym-button-bg);
 color:var(--ym-button-fg);
 border-color:var(--ym-button-bg);
}
.ym-quick-links{
 margin-top:12px;
 padding-top:10px;
 border-top:1px solid var(--ym-divider);
 display:flex;
 flex-wrap:wrap;
 align-items:center;
 gap:8px;
}
.ym-quick-links__label{
 font-size:12px;
 text-transform:uppercase;
 letter-spacing:0.08em;
 color:var(--ym-muted);
}
.ym-action-row{
 margin-top:10px;
 display:flex;
 flex-wrap:wrap;
 gap:10px;
}
.ym-meta-note{
 margin-top:10px;
 font-size:12px;
 opacity:0.8;
}
.ym-noscript{
 margin-top:8px;
}
.ym-noscript a{
 display:block;
 margin:6px 0;
}
@media (max-width:420px){
 .ym-player{max-width:100%;}
 .ym-preview{padding:14px;}
 .ym-play-button{font-size:16px;}
 .ym-meta-panel{padding:16px;}
}
</style>
`)
	if data.AutoplayEnabled && data.AutoplayNextURL != "" {
		fmt.Fprintf(&b, `<meta http-equiv="refresh" content="5;url=%s">`, Escape(data.AutoplayNextURL))
	}
	b.WriteString(`</head><body class="` + ThemeBodyClass(data.Theme) + `">`)
	b.WriteString(RenderHeader("", data.Theme, data.CurrentPath, ""))
	b.WriteString(`<main class="page watch-page">`)
	b.WriteString(`<div class="watch-hero box">`)

	fmt.Fprintf(&b, `<div class="ym-player" id="ym-player" data-stream="%s" data-hls="%s" data-audio="%s" data-lite="%s" data-lite-label="%s" data-preview="%s" data-title="%s" data-meta="%s" data-download-mp4="%s" data-download-3gp="%s">`,
		EscapeAttr(playerStream),
		EscapeAttr(video.HLSManifest),
		EscapeAttr(data.AudioURL),
		EscapeAttr(liteURL),
		EscapeAttr(liteLabel),
		previewAttr,
		EscapeAttr(video.Title),
		EscapeAttr(primaryMeta),
		EscapeAttr(mp4Download),
		EscapeAttr(liteURL),
	)

	fmt.Fprintf(&b, `<div class="ym-preview" id="ym-preview">
<img src="%s" alt="%s" class="ym-preview-img">
<button type="button" id="ym-inline-play" class="ym-play-button">Play<span class="ym-meta">%s</span></button>
<div class="ym-support-hint" id="ym-support-hint">Detecting browser capabilities...</div>
</div>
<noscript>
	<div class="ym-noscript">
		<a class="ym-button ym-button-primary" href="%s">Play in browser</a>
		<a class="ym-button ym-button-ghost" href="%s">Download 3GP / MP4</a>
	</div>
</noscript>
</div>`,
		previewAttr,
		Escape(video.Title),
		Escape(buttonMeta),
		Escape(mp4Download),
		Escape(liteURL),
	)

	authorInitial := ""
	if video.Author != "" {
		runes := []rune(video.Author)
		if len(runes) > 0 {
			authorInitial = strings.ToUpper(string(runes[0]))
		}
	}

	statsParts := []string{}
	if dur := formatVideoDuration(video.LengthSeconds); dur != "" {
		statsParts = append(statsParts, dur)
	}
	if strings.TrimSpace(primaryMeta) != "" {
		statsParts = append(statsParts, primaryMeta)
	}

	watchLaterLabel := ""
	if data.WatchLaterURL != "" {
		if data.InWatchLater {
			watchLaterLabel = "Remove from Watch later"
		} else {
			watchLaterLabel = "Save to Watch later"
		}
	}

	subscribeMarkup := ""
	if data.SubscribeURL != "" {
		subscribeMarkup = `<a class="ym-sub-btn" href="` + Escape(data.SubscribeURL) + `">Subscribe</a>`
	} else if data.UnsubscribeURL != "" {
		subscribeMarkup = `<a class="ym-sub-btn is-on" href="` + Escape(data.UnsubscribeURL) + `">Subscribed</a>`
	}

	b.WriteString(`<div class="ym-meta-panel">`)
	b.WriteString(`<h1 class="ym-title">` + Escape(video.Title) + `</h1>`)
	b.WriteString(`<div class="ym-meta-header">`)
	b.WriteString(`<div class="ym-channel-block">`)
	if authorInitial != "" {
		b.WriteString(`<div class="ym-avatar">` + Escape(authorInitial) + `</div>`)
	}
	b.WriteString(`<div class="ym-channel-details">`)
	if video.Author != "" {
		if video.ChannelID != "" {
			b.WriteString(`<a href="/channel?id=` + Escape(video.ChannelID) + `" class="ym-channel-name">` + Escape(video.Author) + `</a>`)
		} else {
			b.WriteString(`<span class="ym-channel-name">` + Escape(video.Author) + `</span>`)
		}
	}
	if video.ViewCount != "" {
		b.WriteString(`<span class="ym-channel-count">` + Escape(video.ViewCount) + ` total views</span>`)
	}
	b.WriteString(`</div>`)
	b.WriteString(`</div>`)
	if subscribeMarkup != "" {
		b.WriteString(subscribeMarkup)
	}
	b.WriteString(`</div>`)

	if len(statsParts) > 0 {
		b.WriteString(`<div class="ym-meta-stats">`)
		for i, part := range statsParts {
			if i == 0 {
				b.WriteString(`<span>` + Escape(part) + `</span>`)
			} else {
				b.WriteString(`<span>` + Escape(part) + `</span>`)
			}
		}
		b.WriteString(`</div>`)
	}

	if watchLaterLabel != "" || data.AudioURL != "" || len(liteActions) > 0 {
		b.WriteString(`<div class="ym-action-row">`)
		if watchLaterLabel != "" {
			fmt.Fprintf(&b, `<a class="ym-button ym-button-ghost" id="ym-watchlater-link" href="%s">%s</a>`, Escape(data.WatchLaterURL), Escape(watchLaterLabel))
		}
		if data.AudioURL != "" {
			fmt.Fprintf(&b, `<a class="ym-button ym-button-ghost" id="ym-audio-link" href="%s">Audio only (MP3)</a>`, Escape(data.AudioURL))
		}
		for idx, lite := range liteActions {
			linkID := "ym-lite-link"
			if idx == 0 {
				linkID = "ym-lite-link-udp"
			} else if idx == 1 {
				linkID = "ym-lite-link-tcp"
			} else {
				linkID = fmt.Sprintf("ym-lite-link-%d", idx)
			}
			if lite.URL == "" {
				continue
			}
			fmt.Fprintf(&b, `<a class="ym-button ym-button-ghost" id="%s" href="%s">%s</a>`, EscapeAttr(linkID), EscapeAttr(lite.URL), Escape(lite.Label))
		}
		b.WriteString(`</div>`)
	}

	if len(data.TranscodeLinks) > 0 {
		b.WriteString(`<div class="ym-quick-links">`)
		b.WriteString(`<span class="ym-quick-links__label">More formats</span>`)
		for _, link := range data.TranscodeLinks {
			if link.URL == "" {
				continue
			}
			fmt.Fprintf(&b, `<a class="ym-chip" href="%s">%s</a>`, Escape(link.URL), Escape(link.Label))
		}
		b.WriteString(`</div>`)
	}

	b.WriteString(`</div>`)
	b.WriteString(`</div><hr>`)

	if data.AutoplayToggleURL != "" {
		if data.AutoplayEnabled {
			message := "Autoplay is ON"
			if data.AutoplaySource != "" {
				message += " (next: " + Escape(data.AutoplaySource) + ")"
			}
			fmt.Fprintf(&b, `<div class="box"><small>%s - <a href="%s">turn off</a></small></div>`, message, Escape(data.AutoplayToggleURL))
		} else {
			fmt.Fprintf(&b, `<div class="box"><small>Autoplay is OFF - <a href="%s">turn on</a></small></div>`, Escape(data.AutoplayToggleURL))
		}
	}

	if len(data.Queue) > 0 {
		b.WriteString(`<div class="box"><b>Queue</b>`)
		if data.QueueClearURL != "" {
			fmt.Fprintf(&b, ` <small><a href="%s">clear all</a></small>`, Escape(data.QueueClearURL))
		}
		b.WriteString(`<br>`)
		for _, entry := range data.Queue {
			item := entry.Item
			fmt.Fprintf(&b, `<div class="vid">
<table cellspacing="0" cellpadding="2"><tr valign="top">
<td><a href="/watch?v=%s&dequeue=1"><img src="%s" width="96" height="54" alt=""></a></td>
<td><b><a href="/watch?v=%s&dequeue=1">%s</a></b><br>
<small>%s</small><br>
<small><a href="%s">remove</a></small></td>
</tr></table>
</div>`,
				Escape(item.ID), EscapeAttr(ProxiedImage(item.Thumbnail)),
				Escape(item.ID), Escape(item.Title),
				Escape(item.Meta), Escape(entry.RemoveURL))
		}
		b.WriteString(`</div>`)
	}

	if len(data.Related) > 0 {
		b.WriteString(`<div class="box"><b>Related</b></div>`)
		for _, entry := range data.Related {
			item := entry.Item
			channelID := item.ChannelID
			if channelID == "" {
				channelID = item.Channel
			}
			fmt.Fprintf(&b, `<div class="vid">
<table cellspacing="0" cellpadding="2"><tr valign="top">
<td>
	<a href="/watch?v=%s">
		<div style="position:relative;display:inline-block;">
			<img src="%s" width="96" height="54" alt="">
			<div class="badge">%s</div>
		</div>
	</a>
</td>
<td>
	<b><a href="/watch?v=%s">%s</a></b><br>
	<small><a href="/channel?id=%s">%s</a></small><br>
	<small>%s</small><br>
	<small><a href="%s">Add to queue</a></small>
</td>
</tr></table>
</div>`,
				Escape(item.ID), EscapeAttr(ProxiedImage(item.Thumbnail)), Escape(item.Duration),
				Escape(item.ID), Escape(item.Title),
				Escape(channelID), Escape(item.Channel),
				Escape(item.Meta), Escape(entry.AddToQueueURL))
		}
	}

	b.WriteString(`<hr><div class="footer-link"><a href="/">Home</a></div>`)
	b.WriteString(`</main>`)

	b.WriteString(`<script>
function checkBrowserSupport(){
 var probe = document.createElement('video');
 var hasVideo = !!(probe && probe.canPlayType);
 var canMp4 = false;
 var canHls = false;
 if(hasVideo){
  try{
   canMp4 = probe.canPlayType('video/mp4; codecs="avc1.42E01E, mp4a.40.2"') !== '';
   canHls = probe.canPlayType('application/vnd.apple.mpegurl') !== '' || probe.canPlayType('application/x-mpegURL') !== '';
  }catch(e){}
 }
 var hlsJsReady = false;
 if(typeof window.Hls !== 'undefined' && window.Hls && typeof window.Hls.isSupported === 'function'){
  try{ hlsJsReady = window.Hls.isSupported(); }catch(e){}
 }
 if(!canHls && hlsJsReady){
  canHls = true;
 }
 var fetchOk = typeof window.fetch === 'function';
 var mediaSourceOk = typeof window.MediaSource === 'function';
 var domParserOk = typeof window.DOMParser === 'function';
 var ua = '';
 try{ ua = navigator.userAgent || ''; }catch(e){}
 var mini = false;
	if(ua){
	 var low = ua.toLowerCase();
	 if(low.indexOf('opera mini') !== -1 || low.indexOf('ucbrowser') !== -1 || low.indexOf('ucweb') !== -1 || low.indexOf('series40') !== -1 || low.indexOf('s40') !== -1 || low.indexOf('symbian') !== -1 || low.indexOf('netfront') !== -1 || low.indexOf('sonyericsson') !== -1 || low.indexOf('sony ericsson') !== -1){
	  mini = true;
	 }
	 if(low.indexOf('android 2.3') !== -1 || low.indexOf('nokia') !== -1){
	  mini = true;
	 }
	}
 var screenMin = Math.min(screen.width || 0, screen.height || 0);
 if(screenMin && screenMin <= 320){
  mini = true;
 }
 var operaMini = ua && ua.indexOf('Opera Mini') !== -1;
 if(operaMini){
  mini = true;
  hasVideo = false;
 }
 return {
  hasVideo: hasVideo,
  canPlayMp4: canMp4,
  canUseHls: canHls,
  hasFetch: fetchOk,
  hasMediaSource: mediaSourceOk,
  hasDOMParser: domParserOk,
  prefersLite: mini,
  hlsJsReady: hlsJsReady
 };
}

function initVideoPlayer(url){
 var container = document.getElementById('ym-player');
 if(!container || container.getAttribute('data-init') === '1'){
  return;
 }
 var support = checkBrowserSupport();
 var hlsUrl = container.getAttribute('data-hls');
 var preview = container.getAttribute('data-preview');
 var meta = container.getAttribute('data-meta');
 var audio = container.getAttribute('data-audio');
 var lite = container.getAttribute('data-lite');
 if(support.prefersLite){
  renderLiteMode(container, preview, lite || url, audio, meta);
  container.setAttribute('data-init','1');
  return;
 }
 if(support.hasVideo && support.canPlayMp4 && url){
  renderNativeVideo(container, url, preview, meta, support);
  container.setAttribute('data-init','1');
  return;
 }
 if(hlsUrl && support.canUseHls){
  renderHlsPlayer(container, hlsUrl, preview, meta, support);
  container.setAttribute('data-init','1');
  return;
 }
 renderDownloadButtons();
  container.setAttribute('data-init','1');
}

function renderDownloadButtons(){
 var container = document.getElementById('ym-player');
 if(!container){
  return;
 }
 var mp4 = container.getAttribute('data-download-mp4') || container.getAttribute('data-stream') || '';
 var retro = container.getAttribute('data-download-3gp') || '';
 var audio = container.getAttribute('data-audio') || '';
 var html = '<div class="ym-lite-panel"><p>Your browser cannot play the stream inline. Choose an option:</p>';
 if(mp4){
  html += '<a class="ym-button ym-button-primary" href="' + mp4 + '">Play in browser</a>';
 }
 if(retro){
  html += '<a class="ym-button ym-button-ghost" href="' + retro + '">Download 3GP / MP4</a>';
 }
 if(audio){
  html += '<a class="ym-button ym-button-ghost" href="' + audio + '">Audio only</a>';
 }
 html += '</div>';
 container.innerHTML = html;
}

function renderNativeVideo(container, url, poster, meta, support){
 var html = '<video class="ym-video" controls playsinline';
 if(poster){
  html += ' poster="' + poster + '"';
 }
 html += '><source src="' + url + '" type="video/mp4">Your browser does not support HTML5 video.</video>';
 if(meta){
  html += '<div class="ym-meta-note">MP4 - ' + meta + '</div>';
 }
 html += buildSupportDetails(support);
 container.innerHTML = html;
}

function renderHlsPlayer(container, url, poster, meta, support){
 var markup = '<video class="ym-video" controls playsinline';
 if(poster){
  markup += ' poster="' + poster + '"';
 }
 markup += '></video>';
 if(meta){
  markup += '<div class="ym-meta-note">HLS - ' + meta + '</div>';
 }
 markup += buildSupportDetails(support);
 container.innerHTML = markup;
 var videoEl = container.querySelector ? container.querySelector('video') : null;
 if(!videoEl){
  renderDownloadButtons();
  return;
 }
 if(support.hlsJsReady && window.Hls){
  try{
   var hls = new window.Hls();
   hls.loadSource(url);
   hls.attachMedia(videoEl);
  }catch(e){
   videoEl.src = url;
  }
 }else{
  videoEl.src = url;
 }
}

function renderLiteMode(container, preview, lite, audio, meta){
 var html = '<div class="ym-lite-panel">';
 if(preview){
  html += '<img src="' + preview + '" alt="preview">';
 }
 html += '<p>Lite mode for mini browsers.</p>';
 if(meta){
  html += '<div class="ym-meta-note">' + meta + '</div>';
 }
 if(lite){
  html += '<a class="ym-button ym-button-primary" href="' + lite + '">Lite stream</a>';
 }
 if(audio){
  html += '<a class="ym-button ym-button-ghost" href="' + audio + '">Audio only</a>';
 }
 html += '</div>';
 container.innerHTML = html;
}

function buildSupportDetails(support){
 var rows = [];
 rows.push(support.hasVideo ? 'HTML5 video: OK' : 'HTML5 video: missing');
 if(support.canPlayMp4){
  rows.push('MP4: OK');
 }
 if(support.canUseHls){
  rows.push('HLS: OK');
 }
 if(!support.hasFetch){
  rows.push('fetch: missing');
 }
 if(!support.hasMediaSource){
  rows.push('MediaSource: missing');
 }
 if(!support.hasDOMParser){
  rows.push('DOMParser: missing');
 }
 if(!rows.length){
  return '';
 }
 return '<div class="ym-meta-note">' + rows.join(' | ') + '</div>';
}

function onReady(fn){
 if(document.readyState === 'complete' || document.readyState === 'interactive'){
  setTimeout(fn, 0);
  return;
 }
 if(document.addEventListener){
  document.addEventListener('DOMContentLoaded', fn);
 }else if(document.attachEvent){
  document.attachEvent('onreadystatechange', function(){
   if(document.readyState === 'complete'){
    fn();
   }
  });
 }
}

onReady(function(){
 var container = document.getElementById('ym-player');
 if(!container){
  return;
 }
 var stream = container.getAttribute('data-stream');
 var hint = document.getElementById('ym-support-hint');
 var support = checkBrowserSupport();
 if(hint){
  var message = 'Preparing the best playback mode.';
  if(support.prefersLite){
   message = 'Lite mode detected. Use the Lite stream button (RTSP) for best results.';
  }else if(support.canPlayMp4){
   message = 'MP4 playback supported.';
  }else if(support.canUseHls){
   message = 'HLS playback will be used.';
  }else{
   message = 'Video playback unavailable. Use the download buttons.';
  }
  hint.textContent = message;
 }
 var playBtn = document.getElementById('ym-inline-play');
 if(playBtn){
  playBtn.onclick = function(ev){
   if(ev && ev.preventDefault){ ev.preventDefault(); }
   initVideoPlayer(stream);
   return false;
  };
 }
 if(support.hasVideo && support.canPlayMp4){
  initVideoPlayer(stream);
 }
});
</script>
`)

	AppendSuggestionScript(&b)
	b.WriteString(`</body></html>`)
	return b.String()
}

func primaryFormatMeta(formats []youtube.Format) (quality string, size string) {
	for _, f := range formats {
		if f.URL == "" {
			continue
		}
		q := strings.TrimSpace(f.Quality)
		s := formatBytes(f.ContentLength)
		if q != "" || s != "" {
			return q, s
		}
	}
	return "", ""
}

func buildMetaBadge(quality, size string) string {
	quality = strings.TrimSpace(quality)
	size = strings.TrimSpace(size)
	switch {
	case quality == "" && size == "":
		return ""
	case quality == "":
		return size
	case size == "":
		return quality
	default:
		return quality + " - " + size
	}
}

func formatBytes(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil || value <= 0 {
		return ""
	}
	units := []string{"B", "KB", "MB", "GB", "TB"}
	idx := 0
	for value >= 1024 && idx < len(units)-1 {
		value /= 1024
		idx++
	}
	return fmt.Sprintf("%.1f %s", value, units[idx])
}

func formatVideoDuration(length string) string {
	length = strings.TrimSpace(length)
	if length == "" {
		return ""
	}
	secs, err := strconv.Atoi(length)
	if err != nil || secs <= 0 {
		return length
	}
	h := secs / 3600
	m := (secs % 3600) / 60
	s := secs % 60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}

func filterProgressiveFormats(formats []youtube.Format) []youtube.Format {
	out := make([]youtube.Format, 0, len(formats))
	for _, f := range formats {
		if f.URL == "" {
			continue
		}
		if isProgressiveMime(f.Mime) {
			out = append(out, f)
		}
	}
	return out
}

func isProgressiveMime(mime string) bool {
	m := strings.ToLower(mime)
	if strings.Contains(m, "audio") {
		return true
	}
	codecIdx := strings.Index(m, `codecs="`)
	if codecIdx >= 0 {
		rest := m[codecIdx+len(`codecs="`):]
		if strings.Contains(rest, ",") {
			return true
		}
	}
	return strings.Contains(m, "mp2t") || strings.Contains(m, `webm; codecs="vp8.0, vorbis"`)
}

func findLinkByKeyword(links []Link, keywords ...string) (Link, bool) {
	if len(keywords) == 0 {
		return Link{}, false
	}
	for _, link := range links {
		lower := strings.ToLower(link.Label)
		for _, keyword := range keywords {
			if strings.Contains(lower, strings.ToLower(keyword)) {
				return link, true
			}
		}
	}
	return Link{}, false
}

func buildLiteActionLinks(base Link, preferTransport bool) []Link {
	baseURL := strings.TrimSpace(base.URL)
	if baseURL == "" {
		return nil
	}
	label := strings.TrimSpace(base.Label)
	if label == "" {
		label = "Lite stream"
	}
	links := []Link{{Label: label, URL: baseURL}}
	if preferTransport {
		udp := withTransportQuery(baseURL, "udp")
		tcp := withTransportQuery(baseURL, "tcp")
		if udp != "" {
			links = append(links, Link{Label: label + " (UDP)", URL: udp})
		}
		if tcp != "" {
			links = append(links, Link{Label: label + " (TCP)", URL: tcp})
		}
	}
	return links
}

func withTransportQuery(rawURL, transport string) string {
	rawURL = strings.TrimSpace(rawURL)
	transport = strings.TrimSpace(transport)
	if rawURL == "" || transport == "" {
		return rawURL
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	query := parsed.Query()
	query.Set("transport", transport)
	parsed.RawQuery = query.Encode()
	return parsed.String()
}
