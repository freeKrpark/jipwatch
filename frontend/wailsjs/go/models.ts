export namespace main {
	
	export class ComplexCount {
	    aptSeq: string;
	    aptNm: string;
	    umdNm: string;
	    buildYear: number;
	    sales: number;
	
	    static createFrom(source: any = {}) {
	        return new ComplexCount(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.aptSeq = source["aptSeq"];
	        this.aptNm = source["aptNm"];
	        this.umdNm = source["umdNm"];
	        this.buildYear = source["buildYear"];
	        this.sales = source["sales"];
	    }
	}
	export class MonthSummary {
	    total: number;
	    canceled: number;
	    direct: number;
	    complexes: ComplexCount[];
	    fetchedAt: string;
	    compared: boolean;
	    added: number;
	    newlyCanceled: number;
	
	    static createFrom(source: any = {}) {
	        return new MonthSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total = source["total"];
	        this.canceled = source["canceled"];
	        this.direct = source["direct"];
	        this.complexes = this.convertValues(source["complexes"], ComplexCount);
	        this.fetchedAt = source["fetchedAt"];
	        this.compared = source["compared"];
	        this.added = source["added"];
	        this.newlyCanceled = source["newlyCanceled"];
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
	export class SyncResult {
	    months: number;
	    added: number;
	    newlyCanceled: number;
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new SyncResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.months = source["months"];
	        this.added = source["added"];
	        this.newlyCanceled = source["newlyCanceled"];
	        this.error = source["error"];
	    }
	}
	export class WatchItem {
	    aptSeq: string;
	    aptNm: string;
	    sggCd: string;
	    umdNm: string;
	    sales: number;
	    latest?: molit.Trade;
	
	    static createFrom(source: any = {}) {
	        return new WatchItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.aptSeq = source["aptSeq"];
	        this.aptNm = source["aptNm"];
	        this.sggCd = source["sggCd"];
	        this.umdNm = source["umdNm"];
	        this.sales = source["sales"];
	        this.latest = this.convertValues(source["latest"], molit.Trade);
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

export namespace molit {
	
	export class Trade {
	    aptSeq: string;
	    aptNm: string;
	    sggCd: string;
	    umdNm: string;
	    jibun: string;
	    area: number;
	    floor: number;
	    price: number;
	    date: string;
	    direct: boolean;
	    canceled: boolean;
	    canceledOn?: string;
	    registeredOn?: string;
	    buildYear: number;
	
	    static createFrom(source: any = {}) {
	        return new Trade(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.aptSeq = source["aptSeq"];
	        this.aptNm = source["aptNm"];
	        this.sggCd = source["sggCd"];
	        this.umdNm = source["umdNm"];
	        this.jibun = source["jibun"];
	        this.area = source["area"];
	        this.floor = source["floor"];
	        this.price = source["price"];
	        this.date = source["date"];
	        this.direct = source["direct"];
	        this.canceled = source["canceled"];
	        this.canceledOn = source["canceledOn"];
	        this.registeredOn = source["registeredOn"];
	        this.buildYear = source["buildYear"];
	    }
	}

}

