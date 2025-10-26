package ui

// RenderStyle returns the inline CSS used by the retro UI.
func RenderStyle() string {
	return `
body{font-family:Arial,sans-serif;font-size:13px;line-height:1.45;margin:0;background:#f9f9f9;color:#111;}
body.theme-dark{background:#181818;color:#fff;}
a{color:#c00;text-decoration:none;}
body.theme-dark a{color:#ff4e45;}
a:hover{text-decoration:underline;}
header,nav,section,article,footer,main{display:block;}
.head{background:#fff;border-bottom:1px solid #e5e5e5;padding:8px 10px;overflow:hidden;zoom:1;}
body.theme-dark .head{background:#202020;border-bottom:1px solid #303030;}
.head-logo{float:left;font-weight:bold;font-size:16px;color:#111;margin:4px 12px 4px 0;text-decoration:none;}
body.theme-dark .head-logo{color:#fff;}
.head-logo img{display:block;height:24px;width:auto;}
.head-search{float:left;border:1px solid #e5e5e5;-webkit-border-radius:18px;border-radius:18px;background:#fff;padding:2px;margin:0 12px 0 0;}
body.theme-dark .head-search{border:1px solid #303030;background:#202020;}
.head-search .search-input{float:left;border:0;background:transparent;color:#111;padding:4px 8px;min-width:160px;font-size:13px;}
body.theme-dark .head-search .search-input{color:#fff;}
.head-search .search-submit{float:left;border:0;background:#c00;color:#fff;padding:4px 12px;font-weight:bold;font-size:12px;-webkit-border-radius:16px;border-radius:16px;cursor:pointer;}
body.theme-dark .head-search .search-submit{background:#ff4e45;}
.head-actions{float:right;font-size:12px;margin-top:6px;color:#606060;}
body.theme-dark .head-actions{color:#aaa;}
.head-theme{color:inherit;text-decoration:none;}
.head-theme:hover{text-decoration:underline;}
.tabs{background:#e5e5e5;border-bottom:1px solid #e5e5e5;padding:6px 0;text-align:center;}
body.theme-dark .tabs{background:#303030;border-bottom:1px solid #303030;}
.tabs a{display:inline-block;padding:6px 12px;margin:0 2px;background:#fff;color:#606060;font-size:12px;-webkit-border-radius:14px;border-radius:14px;}
body.theme-dark .tabs a{background:#202020;color:#aaa;}
.tabs a.on{background:#c00;color:#fff;font-weight:bold;}
body.theme-dark .tabs a.on{background:#ff4e45;}
.page{width:94%;margin:0 auto;padding:16px 0 48px;}
.footer-link{text-align:center;margin:24px 0;font-weight:bold;}
.footer-link a{color:#c00;}
body.theme-dark .footer-link a{color:#ff4e45;}
.box{background:#fff;margin:12px 0;padding:14px;-webkit-border-radius:8px;border-radius:8px;-webkit-box-shadow:0 1px 2px rgba(0,0,0,0.08);box-shadow:0 1px 2px rgba(0,0,0,0.08);}
body.theme-dark .box{background:#202020;-webkit-box-shadow:0 1px 2px rgba(0,0,0,0.4);box-shadow:0 1px 2px rgba(0,0,0,0.4);}
.channel-tabs{margin:10px 0;}
.channel-tabs a{display:inline-block;padding:6px 10px;border:1px solid #e5e5e5;-webkit-border-radius:14px;border-radius:14px;margin:0 4px 4px 0;color:#606060;background:#fff;}
body.theme-dark .channel-tabs a{border-color:#303030;background:#202020;color:#aaa;}
.channel-tabs a.on{background:#c00;color:#fff;border-color:#c00;}
body.theme-dark .channel-tabs a.on{background:#ff4e45;border-color:#ff4e45;}
.watch-hero{text-align:center;}
.watch-hero img{width:240px;max-width:100%;-webkit-border-radius:12px;border-radius:12px;}
.feed-card{background:#fff;border-bottom:1px solid #e5e5e5;padding:12px 8px;position:relative;zoom:1;}
.feed-card:after{content:"";display:block;clear:both;height:0;}
.feed-card:hover{background:#f3f3f3;}
body.theme-dark .feed-card{background:#202020;border-bottom:1px solid #303030;}
body.theme-dark .feed-card:hover{background:#2a2a2a;}
.feed-thumb{float:left;width:168px;position:relative;margin-right:12px;}
.feed-thumb img{width:100%;height:auto;-webkit-border-radius:10px;border-radius:10px;background:#000;display:block;}
.badge{position:absolute;bottom:6px;right:6px;background:rgba(0,0,0,0.75);color:#fff;font-size:10px;padding:2px 4px;-webkit-border-radius:4px;border-radius:4px;}
.feed-body{margin-left:180px;}
.feed-title{margin:0;font-size:14px;line-height:1.4;font-weight:bold;}
.feed-title a{color:#111;}
body.theme-dark .feed-title a{color:#fff;}
.feed-channel{font-size:12px;color:#606060;margin-top:4px;}
body.theme-dark .feed-channel{color:#aaa;}
.feed-channel a{color:inherit;text-decoration:none;}
.feed-channel a:hover{text-decoration:underline;}
.feed-meta{font-size:12px;color:#606060;margin-top:4px;}
body.theme-dark .feed-meta{color:#aaa;}
.feed-actions{font-size:11px;margin-top:6px;color:#c00;}
body.theme-dark .feed-actions{color:#ff4e45;}
.feed-actions a{color:inherit;text-decoration:none;}
.feed-actions a:hover{text-decoration:underline;}
.feed-dot{margin:0 4px;color:#606060;}
body.theme-dark .feed-dot{color:#aaa;}
.vid{border-bottom:1px solid #e5e5e5;padding:8px 0;}
body.theme-dark .vid{border-bottom:1px solid #303030;}
.vid img{width:168px;height:94px;background:#000;-webkit-border-radius:6px;border-radius:6px;}
small{color:#606060;}
body.theme-dark small{color:#aaa;}
 a.btn{display:inline-block;background:#c00;color:#fff;padding:8px 16px;margin:12px 0;-webkit-border-radius:18px;border-radius:18px;font-weight:bold;}
 body.theme-dark a.btn{background:#ff4e45;color:#fff;}
a.btn:active{opacity:0.8;}
hr{border:0;border-top:1px solid #e5e5e5;margin:24px 0;}
body.theme-dark hr{border-top:1px solid #303030;}
.empty{text-align:center;color:#606060;padding:24px;}
body.theme-dark .empty{color:#aaa;}
table{width:100%;border-collapse:collapse;}
ul{margin:6px 0 0 16px;padding:0;}
`
}
