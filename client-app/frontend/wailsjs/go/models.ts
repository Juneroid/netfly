export namespace frpclient {
	
	export class ProxyState {
	    name: string;
	    type: string;
	    status: string;
	    err: string;
	    remoteAddr: string;
	
	    static createFrom(source: any = {}) {
	        return new ProxyState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.type = source["type"];
	        this.status = source["status"];
	        this.err = source["err"];
	        this.remoteAddr = source["remoteAddr"];
	    }
	}

}

export namespace main {
	
	export class NodeNetStat {
	    name: string;
	    trafficIn: number;
	    trafficOut: number;
	    rateIn: number;
	    rateOut: number;
	
	    static createFrom(source: any = {}) {
	        return new NodeNetStat(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.trafficIn = source["trafficIn"];
	        this.trafficOut = source["trafficOut"];
	        this.rateIn = source["rateIn"];
	        this.rateOut = source["rateOut"];
	    }
	}
	export class NetworkStats {
	    connState: string;
	    nodes: NodeNetStat[];
	    totalIn: number;
	    totalOut: number;
	
	    static createFrom(source: any = {}) {
	        return new NetworkStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.connState = source["connState"];
	        this.nodes = this.convertValues(source["nodes"], NodeNetStat);
	        this.totalIn = source["totalIn"];
	        this.totalOut = source["totalOut"];
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

export namespace model {
	
	export class ProxyItem {
	    name: string;
	    type: string;
	    localIP: string;
	    localPort: number;
	    remotePort: number;
	    customDomains: string;
	    subDomain: string;
	    useEncryption: boolean;
	    useCompression: boolean;
	    enabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ProxyItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.type = source["type"];
	        this.localIP = source["localIP"];
	        this.localPort = source["localPort"];
	        this.remotePort = source["remotePort"];
	        this.customDomains = source["customDomains"];
	        this.subDomain = source["subDomain"];
	        this.useEncryption = source["useEncryption"];
	        this.useCompression = source["useCompression"];
	        this.enabled = source["enabled"];
	    }
	}
	export class ClientAppConfig {
	    serverAddr: string;
	    serverPort: number;
	    token: string;
	    user: string;
	    logLevel: string;
	    autoStart: boolean;
	    proxies: ProxyItem[];
	
	    static createFrom(source: any = {}) {
	        return new ClientAppConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.serverAddr = source["serverAddr"];
	        this.serverPort = source["serverPort"];
	        this.token = source["token"];
	        this.user = source["user"];
	        this.logLevel = source["logLevel"];
	        this.autoStart = source["autoStart"];
	        this.proxies = this.convertValues(source["proxies"], ProxyItem);
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

