export namespace domain {
	
	export class Progress {
	    downloaded_bytes: number;
	    total_bytes: number;
	    percent: number;
	    speed: number;
	
	    static createFrom(source: any = {}) {
	        return new Progress(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.downloaded_bytes = source["downloaded_bytes"];
	        this.total_bytes = source["total_bytes"];
	        this.percent = source["percent"];
	        this.speed = source["speed"];
	    }
	}
	export class DownloadTask {
	    id: string;
	    url: string;
	    filename: string;
	    dest_path: string;
	    total_size: number;
	    progress: Progress;
	    status: string;
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new DownloadTask(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.url = source["url"];
	        this.filename = source["filename"];
	        this.dest_path = source["dest_path"];
	        this.total_size = source["total_size"];
	        this.progress = this.convertValues(source["progress"], Progress);
	        this.status = source["status"];
	        this.error = source["error"];
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

