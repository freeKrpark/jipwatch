export namespace main {
	
	export class Mark {
	    month: string;
	    price: number;
	
	    static createFrom(source: any = {}) {
	        return new Mark(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.month = source["month"];
	        this.price = source["price"];
	    }
	}
	export class Summary {
	    peak?: Mark;
	    trough?: Mark;
	    recentMedian: number;
	    recentSales: number;
	    recentFrom: string;
	    fromPeak: number;
	    fromTrough: number;
	
	    static createFrom(source: any = {}) {
	        return new Summary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.peak = this.convertValues(source["peak"], Mark);
	        this.trough = this.convertValues(source["trough"], Mark);
	        this.recentMedian = source["recentMedian"];
	        this.recentSales = source["recentSales"];
	        this.recentFrom = source["recentFrom"];
	        this.fromPeak = source["fromPeak"];
	        this.fromTrough = source["fromTrough"];
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
	export class TrendPoint {
	    month: string;
	    median: number;
	    sales: number;
	
	    static createFrom(source: any = {}) {
	        return new TrendPoint(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.month = source["month"];
	        this.median = source["median"];
	        this.sales = source["sales"];
	    }
	}
	export class Classified {
	    trade: molit.Trade;
	    reason: string;
	
	    static createFrom(source: any = {}) {
	        return new Classified(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.trade = this.convertValues(source["trade"], molit.Trade);
	        this.reason = source["reason"];
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
	export class TypeCount {
	    area: number;
	    sales: number;
	
	    static createFrom(source: any = {}) {
	        return new TypeCount(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.area = source["area"];
	        this.sales = source["sales"];
	    }
	}
	export class Analysis {
	    types: TypeCount[];
	    area: number;
	    trades: Classified[];
	    trend: TrendPoint[];
	    summary: Summary;
	
	    static createFrom(source: any = {}) {
	        return new Analysis(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.types = this.convertValues(source["types"], TypeCount);
	        this.area = source["area"];
	        this.trades = this.convertValues(source["trades"], Classified);
	        this.trend = this.convertValues(source["trend"], TrendPoint);
	        this.summary = this.convertValues(source["summary"], Summary);
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
	export class Coverage {
	    from: string;
	    fetched: number;
	    total: number;
	
	    static createFrom(source: any = {}) {
	        return new Coverage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.from = source["from"];
	        this.fetched = source["fetched"];
	        this.total = source["total"];
	    }
	}
	export class District {
	    code: string;
	    name: string;
	
	    static createFrom(source: any = {}) {
	        return new District(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.code = source["code"];
	        this.name = source["name"];
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

