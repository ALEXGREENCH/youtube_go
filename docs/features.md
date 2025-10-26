# YouTube Mini Feature Matrix

This matrix captures standard YouTube features, their current status inside the retro frontend, and the module that will own them.

## Discovery & Navigation
- Home recommendations: Done (`internal/features/home`)
- Trending / Explore: Done (`internal/features/explore`)
- Search: Done (`internal/features/search`)
- Unified home/search layout: Done (`internal/features/home`)
- Channel navigation: Done (`internal/features/channel`)
- Playlist browsing: Stub (`internal/features/playlist`)
- Shorts shelf: Planned (`internal/features/shorts` planned module)
- Topic / hashtag hubs: Planned (`internal/features/topics` planned module)

## Playback Experience
- Video playback (multiple qualities): Done (`internal/features/watch`)
- Audio-only / background mode: Done (`internal/features/audio`)
- Captions and subtitles: Planned (`internal/features/captions` planned module)
- Related / up-next videos: Done (`internal/features/watch`)
- Autoplay toggle: Done (`internal/features/watch`)
- Playback queue: Done (`internal/features/watch`)
- Live streams and premieres: Planned (`internal/features/live` planned module)
- In-player actions (like/share/report): Deferred (requires auth flows)

## Social & Interaction
- Comments and replies: Planned (`internal/features/comments` planned module)
- Comment moderation filters: Planned (`internal/features/comments` planned module)
- Likes / dislikes: Deferred (requires authenticated calls)
- Subscriptions feed: Done (`internal/features/subscriptions`)
- Notifications bell: Deferred (needs push + auth)
- Community posts: Planned (`internal/features/community` planned module)

## Library & Personalisation
- Watch history: Done (`internal/features/history`)
- Watch later: Done (`internal/features/watchlater`)
- Playlist management: Deferred (write access requires auth)
- Downloads: Planned (`internal/features/downloads` planned module)
- Subscriptions highlights: Planned (`internal/features/subscriptions`)

## Account & Settings
- Sign-in / Google account integration: Deferred (scope + compliance)
- Profile and channel switcher: Planned (`internal/features/account` planned module)
- Language and region settings: Planned (`internal/features/settings` planned module)
- Parental controls / restricted mode: Planned (`internal/features/settings` planned module)

## Platform Enhancements
- Search suggestions (typeahead): Done (`internal/features/suggest`)
- Offline-first caching: Done (`internal/platform/cache`)
- API client with quota/backoff: Done (`internal/youtube`)
- Observability (metrics/logging): Done (`internal/platform/metrics`)
- Responsive retro UI theme: Done (`internal/ui`)
- Light/Dark theme toggle: Done (`internal/features/theme`)
- Media proxy for thumbnails/assets: Done (`internal/features/proxy`)
- Legacy device transcoding: Done (`internal/transcode`)

### Reference
- Feature status page: GET `/features`
- Main entrypoint: `cmd/youtube-mini`
- Router wiring: `internal/app`
- Feature handlers: `internal/features/*`
- YouTube API adapter: `internal/youtube`
- Shared utilities and caching: `internal/platform/*`
- UI helpers and styles: `internal/ui`
- FFmpeg service: `internal/transcode`

Status keywords: Done = functional, Stub = routed with placeholder UI, Planned = defined roadmap module, Deferred = blocked on external dependencies.

