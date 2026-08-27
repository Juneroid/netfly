export namespace frpserver {
	
	export class ProxyBrief {
	    name: string;
	    type: string;
	    user: string;
	    curConns: number;
	    todayTrafficIn: number;
	    todayTrafficOut: number;
	
	    static createFrom(source: any = {}) {
	        return new ProxyBrief(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.type = source["type"];
	        this.user = source["user"];
	        this.curConns = source["curConns"];
	        this.todayTrafficIn = source["todayTrafficIn"];
	        this.todayTrafficOut = source["todayTrafficOut"];
	    }
	}
	export class Stats {
	    running: boolean;
	    clientCount: number;
	    proxyCount: number;
	    proxyTypeCounts: Record<string, number>;
	    curConns: number;
	    sessionTrafficIn: number;
	    sessionTrafficOut: number;
	
	    static createFrom(source: any = {}) {
	        return new Stats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.running = source["running"];
	        this.clientCount = source["clientCount"];
	        this.proxyCount = source["proxyCount"];
	        this.proxyTypeCounts = source["proxyTypeCounts"];
	        this.curConns = source["curConns"];
	        this.sessionTrafficIn = source["sessionTrafficIn"];
	        this.sessionTrafficOut = source["sessionTrafficOut"];
	    }
	}

}

export namespace model {
	
	export class ServerAppConfig {
	    bindAddr: string;
	    bindPort: number;
	    token: string;
	    vhostHTTPPort: number;
	    vhostHTTPSPort: number;
	    dashboardEnabled: boolean;
	    dashboardPort: number;
	    dashboardUser: string;
	    dashboardPassword: string;
	    logLevel: string;
	    autoStart: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ServerAppConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.bindAddr = source["bindAddr"];
	        this.bindPort = source["bindPort"];
	        this.token = source["token"];
	        this.vhostHTTPPort = source["vhostHTTPPort"];
	        this.vhostHTTPSPort = source["vhostHTTPSPort"];
	        this.dashboardEnabled = source["dashboardEnabled"];
	        this.dashboardPort = source["dashboardPort"];
	        this.dashboardUser = source["dashboardUser"];
	        this.dashboardPassword = source["dashboardPassword"];
	        this.logLevel = source["logLevel"];
	        this.autoStart = source["autoStart"];
	    }
	}

}

