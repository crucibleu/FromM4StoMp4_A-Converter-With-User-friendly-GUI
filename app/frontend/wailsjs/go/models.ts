export namespace engine {
	
	export class Comment {
	    t: number;
	    m: number;
	    s: number;
	    c: number;
	    text: string;
	
	    static createFrom(source: any = {}) {
	        return new Comment(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.t = source["t"];
	        this.m = source["m"];
	        this.s = source["s"];
	        this.c = source["c"];
	        this.text = source["text"];
	    }
	}
	export class Job {
	    id: string;
	    videoId: string;
	    series: string;
	    part: string;
	    size: number;
	    output: string;
	    state: string;
	    stage: string;
	    progress: number;
	    error: string;
	    started: number;
	    finished: number;
	
	    static createFrom(source: any = {}) {
	        return new Job(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.videoId = source["videoId"];
	        this.series = source["series"];
	        this.part = source["part"];
	        this.size = source["size"];
	        this.output = source["output"];
	        this.state = source["state"];
	        this.stage = source["stage"];
	        this.progress = source["progress"];
	        this.error = source["error"];
	        this.started = source["started"];
	        this.finished = source["finished"];
	    }
	}
	export class MediaInfo {
	    kind: string;
	    codec: string;
	    width: number;
	    height: number;
	    duration: number;
	    hasVideo: boolean;
	    hasAudio: boolean;
	
	    static createFrom(source: any = {}) {
	        return new MediaInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.codec = source["codec"];
	        this.width = source["width"];
	        this.height = source["height"];
	        this.duration = source["duration"];
	        this.hasVideo = source["hasVideo"];
	        this.hasAudio = source["hasAudio"];
	    }
	}
	export class Video {
	    id: string;
	    dir: string;
	    source: string;
	    series: string;
	    part: string;
	    page: number;
	    uploader: string;
	    bvid: string;
	    aid: string;
	    cid: string;
	    videoFile: string;
	    audioFile: string;
	    size: number;
	    duration: number;
	    video: MediaInfo;
	    audio: MediaInfo;
	    coverFile: string;
	    coverUrl: string;
	    danmakuFile: string;
	    progress: number;
	    status: string;
	    output: string;
	    note: string;
	
	    static createFrom(source: any = {}) {
	        return new Video(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.dir = source["dir"];
	        this.source = source["source"];
	        this.series = source["series"];
	        this.part = source["part"];
	        this.page = source["page"];
	        this.uploader = source["uploader"];
	        this.bvid = source["bvid"];
	        this.aid = source["aid"];
	        this.cid = source["cid"];
	        this.videoFile = source["videoFile"];
	        this.audioFile = source["audioFile"];
	        this.size = source["size"];
	        this.duration = source["duration"];
	        this.video = this.convertValues(source["video"], MediaInfo);
	        this.audio = this.convertValues(source["audio"], MediaInfo);
	        this.coverFile = source["coverFile"];
	        this.coverUrl = source["coverUrl"];
	        this.danmakuFile = source["danmakuFile"];
	        this.progress = source["progress"];
	        this.status = source["status"];
	        this.output = source["output"];
	        this.note = source["note"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace main {
	
	export class Settings {
	    cacheDir: string;
	    outputDir: string;
	    template: string;
	    danmaku: boolean;
	    downloadDanmaku: boolean;
	    overwrite: boolean;
	    workers: number;
	    effects: string;
	    language: string;
	    mp4box: string;
	    acceptedTerms: boolean;
	    danmakuOverlay: boolean;
	    danmakuOpacity: number;
	    danmakuDensity: number;
	    danmakuSize: number;
	    volume: number;
	
	    static createFrom(source: any = {}) {
	        return new Settings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.cacheDir = source["cacheDir"];
	        this.outputDir = source["outputDir"];
	        this.template = source["template"];
	        this.danmaku = source["danmaku"];
	        this.downloadDanmaku = source["downloadDanmaku"];
	        this.overwrite = source["overwrite"];
	        this.workers = source["workers"];
	        this.effects = source["effects"];
	        this.language = source["language"];
	        this.mp4box = source["mp4box"];
	        this.acceptedTerms = source["acceptedTerms"];
	        this.danmakuOverlay = source["danmakuOverlay"];
	        this.danmakuOpacity = source["danmakuOpacity"];
	        this.danmakuDensity = source["danmakuDensity"];
	        this.danmakuSize = source["danmakuSize"];
	        this.volume = source["volume"];
	    }
	}
	export class InitData {
	    settings: Settings;
	    outputDir: string;
	    version: string;
	    platform: string;
	    mediaBase: string;
	    mediaToken: string;
	    detectedDirs: string[];
	    defaultTemplate: string;
	
	    static createFrom(source: any = {}) {
	        return new InitData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.settings = this.convertValues(source["settings"], Settings);
	        this.outputDir = source["outputDir"];
	        this.version = source["version"];
	        this.platform = source["platform"];
	        this.mediaBase = source["mediaBase"];
	        this.mediaToken = source["mediaToken"];
	        this.detectedDirs = source["detectedDirs"];
	        this.defaultTemplate = source["defaultTemplate"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class PlayState {
	    position: number;
	    watched: boolean;
	
	    static createFrom(source: any = {}) {
	        return new PlayState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.position = source["position"];
	        this.watched = source["watched"];
	    }
	}
	export class LibraryItem {
	    path: string;
	    name: string;
	    folder: string;
	    size: number;
	    modified: number;
	    info: engine.MediaInfo;
	    hasDanmaku: boolean;
	    play: PlayState;
	
	    static createFrom(source: any = {}) {
	        return new LibraryItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.name = source["name"];
	        this.folder = source["folder"];
	        this.size = source["size"];
	        this.modified = source["modified"];
	        this.info = this.convertValues(source["info"], engine.MediaInfo);
	        this.hasDanmaku = source["hasDanmaku"];
	        this.play = this.convertValues(source["play"], PlayState);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class ScanResult {
	    cacheDir: string;
	    outputDir: string;
	    videos: engine.Video[];
	    elapsed: number;
	
	    static createFrom(source: any = {}) {
	        return new ScanResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.cacheDir = source["cacheDir"];
	        this.outputDir = source["outputDir"];
	        this.videos = this.convertValues(source["videos"], engine.Video);
	        this.elapsed = source["elapsed"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

