import Chart from 'chart.js/auto';
import {
    BuyRent, BuyRentStart, ComplexAnalysis, ComplexTrades, Coverage, DefaultMonth, Districts, HasAPIKey,
    IsWatched, Month, RefreshMonth, SaveAPIKey, SyncWatched, Unwatch, Watch, Watchlist,
} from '../wailsjs/go/main/App';
import {EventsOn} from '../wailsjs/runtime/runtime';

const app = document.getElementById('app');
const nav = document.getElementById('nav');
const syncBar = document.getElementById('sync');

// h builds an element. Strings become text nodes, so names from the API can
// never be read as HTML.
function h(tag, attrs = {}, ...children) {
    const el = document.createElement(tag);
    for (const [k, v] of Object.entries(attrs)) {
        if (k.startsWith('on')) el.addEventListener(k.slice(2), v);
        else if (v !== false && v != null) el.setAttribute(k, v === true ? '' : v);
    }
    el.append(...children.flat().filter((c) => c != null && c !== false));
    return el;
}

// show replaces the page. Like h, it skips false and null, so a node can be
// written as `condition && node`.
function show(page, ...nodes) {
    nav.hidden = page === 'setup';
    for (const tab of nav.querySelectorAll('.tab')) tab.classList.toggle('active', tab.dataset.page === page);
    app.replaceChildren(...nodes.filter((n) => n != null && n !== false));
}

// errorText turns a rejected Go call (a string) or a JS error into a message.
function errorText(err) {
    return typeof err === 'string' ? err : (err && err.message) || String(err);
}

// busy disables a button while a Go call runs and returns the call's result.
async function busy(button, label, work) {
    const before = button.textContent;
    button.disabled = true;
    button.textContent = label;
    try {
        return await work();
    } finally {
        button.disabled = false;
        button.textContent = before;
    }
}

const eok = (manwon) => (manwon / 10000).toFixed(2) + '억';
const monthLabel = (ym) => `${ym.slice(0, 4)}년 ${Number(ym.slice(4))}월`;

// districts is filled from Go at start; gu turns a district code into its name.
let districts = [];
const gu = (code) => (districts.find((d) => d.code === code) || {name: code}).name;

// registeredSince is the first contract date the API reports registration
// for; older sales have no 등기일 at all, so blank says nothing about them.
const registeredSince = '2023-01-01';

// reasonLabels names why a sale is left out of the price analysis.
const reasonLabels = {canceled: '해제', direct: '직거래', low_floor: '1~2층'};

// flags lists why a sale stands out: canceled, sold without a broker, low
// floor, or not yet registered.
function flags({trade: t, reason}) {
    return [
        t.canceled && h('span', {class: 'flag warn', title: `해제일 ${t.canceledOn || ''}`},
            t.canceledOn ? `해제 ${t.canceledOn.slice(5)}` : '해제'),
        t.direct && h('span', {class: 'flag warn'}, '직거래'),
        reason === 'low_floor' && h('span', {class: 'flag'}, '1~2층'),
        !t.canceled && !t.registeredOn && t.date >= registeredSince && h('span', {class: 'flag'}, '미등기'),
    ];
}

// asRows marks sales for tradeTable when no unit type was analysed: only
// canceled and direct sales are set apart.
const asRows = (trades) => trades.map((t) => ({trade: t, reason: t.canceled ? 'canceled' : t.direct ? 'direct' : ''}));

// tradeTable lists sales; rows left out of the analysis are dimmed.
function tradeTable(rows) {
    return h('div', {class: 'table'}, h('table', {},
        h('thead', {}, h('tr', {},
            h('th', {}, '계약일'), h('th', {class: 'num'}, '전용㎡'), h('th', {class: 'num'}, '층'),
            h('th', {class: 'num'}, '금액'), h('th', {}, '표시'))),
        h('tbody', {}, rows.map((r) => h('tr', {class: r.reason ? 'dim' : false},
            h('td', {}, r.trade.date),
            h('td', {class: 'num'}, r.trade.area.toFixed(2)),
            h('td', {class: 'num'}, String(r.trade.floor)),
            h('td', {class: 'num'}, eok(r.trade.price)),
            h('td', {class: 'wrap'}, flags(r)),
        ))),
    ));
}

// --- chart -------------------------------------------------------------------

// charts on the page, destroyed before the page is redrawn.
let charts = [];

function clearCharts() {
    for (const c of charts) c.destroy();
    charts = [];
}

// yearTicks returns 1 January of every year from the first x through the
// year after the last, so an x axis can tick on whole years.
function yearTicks(xs) {
    if (!xs.length) return [];
    const first = new Date(Math.min(...xs)).getUTCFullYear();
    const last = new Date(Math.max(...xs)).getUTCFullYear();
    const years = [];
    for (let y = first; y <= last + 1; y++) years.push(Date.UTC(y, 0, 1));
    return years;
}

// yearAxis is an x axis over years that ticks on each 1 January.
const yearAxis = (years, color) => ({
    type: 'linear', min: years[0], max: years[years.length - 1],
    afterBuildTicks: (axis) => { axis.ticks = years.map((value) => ({value})); },
    ticks: {color: color('--muted'), callback: (v) => new Date(v).getUTCFullYear()},
    grid: {color: color('--line')},
});

// trendRadius matches the Go constant: points read off a wider window are
// drawn dashed, since sales were too thin for the usual one.
const trendRadius = 3;
const isWide = (pt) => Boolean(pt && pt.raw && pt.raw.trend && pt.raw.trend.radius > trendRadius);

const monthMs = (month) => Date.UTC(Number(month.slice(0, 4)), Number(month.slice(5, 7)) - 1, 15);
const markLabel = (m) => `${eok(m.price)} (${m.month.replace('-', '.')})`;
const signed = (v) => `${v > 0 ? '+' : ''}${v.toFixed(1)}%`;

// drawChart plots one unit type's sales over time: sales the analysis uses
// in the accent colour, direct and low-floor sales in grey, the rolling
// median as a line (dashed where it had to widen), and the peak, trough and
// recent median across. Canceled sales are left off; most are re-reported
// and would show twice.
function drawChart(canvas, a) {
    const rows = a.trades;
    const s = a.summary;
    const css = getComputedStyle(document.documentElement);
    const color = (name) => css.getPropertyValue(name).trim();
    const point = (r) => ({x: Date.parse(r.trade.date), y: r.trade.price / 10000, row: r});
    const used = rows.filter((r) => !r.reason).map(point);
    const left = rows.filter((r) => r.reason && r.reason !== 'canceled').map(point);
    const years = yearTicks(used.concat(left).map((p) => p.x));
    const across = (label, price, colour) => ({
        type: 'line', label, borderColor: colour, borderWidth: 1, borderDash: [6, 4], pointRadius: 0,
        data: [{x: years[0], y: price / 10000}, {x: years[years.length - 1], y: price / 10000}],
    });
    const datasets = [
        {label: '분석에 쓰는 거래', data: used, backgroundColor: color('--accent'), pointRadius: 3.5},
        {label: '제외 (직거래·1~2층)', data: left, backgroundColor: color('--muted'), pointRadius: 3, pointStyle: 'rectRot'},
        {
            type: 'line', label: '추세 (앞뒤 3개월 중간값)', borderColor: color('--fg'), borderWidth: 2,
            pointRadius: 0, tension: 0.3, spanGaps: false,
            segment: {borderDash: (ctx) => (isWide(ctx.p0) || isWide(ctx.p1) ? [5, 4] : undefined)},
            data: a.trend.map((p) => ({x: monthMs(p.month), y: p.median ? p.median / 10000 : null, trend: p})),
        },
    ];
    if (s.peak) datasets.push(across(`고점 ${markLabel(s.peak)}`, s.peak.price, color('--muted')));
    if (s.trough) datasets.push(across(`바닥 ${markLabel(s.trough)}`, s.trough.price, color('--warn')));
    if (s.recentMedian) datasets.push(across(`최근 ${eok(s.recentMedian)}`, s.recentMedian, color('--accent')));

    charts.push(new Chart(canvas, {
        type: 'scatter',
        data: {datasets},
        options: {
            animation: false,
            maintainAspectRatio: false,
            scales: {
                x: yearAxis(years, color),
                y: {
                    ticks: {color: color('--muted'), callback: (v) => `${v}억`},
                    grid: {color: color('--line')},
                },
            },
            plugins: {
                legend: {labels: {color: color('--muted'), usePointStyle: true, boxHeight: 6}},
                tooltip: {
                    callbacks: {
                        label: (ctx) => {
                            const raw = ctx.raw;
                            if (raw.trend) {
                                const p = raw.trend;
                                return `${p.month} 추세 ${eok(p.median)} (앞뒤 ${p.radius}개월 ${p.sales}건)`;
                            }
                            if (!raw.row) return ctx.dataset.label;
                            const {trade: t, reason} = raw.row;
                            const why = reason ? ` · ${reasonLabels[reason]}` : '';
                            return `${t.date} · ${eok(t.price)} · ${t.floor}층${why}`;
                        },
                    },
                },
            },
        },
    }));
}

// verdict reads a premium's distance from its usual level in standard
// deviations: past one either way, it is out of the complex's usual range.
function verdict(r) {
    if (!r.enough) return {text: '비교할 기간이 짧아요', tone: ''};
    if (r.z >= 1) return {text: '평소보다 비싸게 거래 중', tone: 'hot'};
    if (r.z <= -1) return {text: '평소보다 싸게 거래 중', tone: 'cool'};
    return {text: '평소 수준', tone: ''};
}

// markNote dates a peak or trough, and warns when it came from a widened
// window: thin sales smooth it, so the real swing was likely sharper.
const markNote = (m) => monthLabel(m.month.replace('-', '')) + (m.wide ? ' · 거래가 적어 넓게 본 값' : '');

// summaryCards leads the complex screen: where recent prices sit against
// the peak and trough of the trend line, and against the district.
function summaryCards(a) {
    const s = a.summary;
    const r = a.relative;
    const v = verdict(r);
    const card = (label, value, note, tone = '') => h('div', {class: `card ${tone}`},
        h('span', {class: 'small'}, label), h('strong', {}, value), note && h('span', {class: 'small'}, note));
    return h('div', {class: 'cards'},
        card('최근 6개월 중간값', s.recentMedian ? eok(s.recentMedian) : '거래 없음',
            `${s.recentSales}건 · ${s.recentFrom} 이후`),
        card('하락 전 고점', s.peak ? eok(s.peak.price) : '—', s.peak && markNote(s.peak)),
        card('하락기 바닥', s.trough ? eok(s.trough.price) : '—', s.trough && markNote(s.trough)),
        card('지금 위치',
            s.recentMedian && s.trough ? `바닥 ${signed(s.fromTrough)}` : '—',
            s.recentMedian && s.peak ? `${s.fromPeak > 0 ? '전고점 넘음' : '고점'} ${signed(s.fromPeak)}` : null),
        card('구 대비',
            r.current ? signed(r.current) : '—',
            r.points.length ? `평소 ${signed(r.usual)} · ${v.text}` : '비교할 거래가 없어요', v.tone),
    );
}

// drawPremium plots how far above the district's same-size sales the
// complex traded, month by month, with its usual premium and usual range.
function drawPremium(canvas, r) {
    const css = getComputedStyle(document.documentElement);
    const color = (name) => css.getPropertyValue(name).trim();
    const years = yearTicks(r.points.map((p) => monthMs(p.month)));
    const flat = (y) => [{x: years[0], y}, {x: years[years.length - 1], y}];
    charts.push(new Chart(canvas, {
        type: 'line',
        data: {
            datasets: [
                // order: lower draws on top, so the band stays behind the lines.
                {label: '프리미엄', order: 0, data: r.points.map((p) => ({x: monthMs(p.month), y: p.premium})),
                    borderColor: color('--fg'), borderWidth: 2, pointRadius: 0, tension: 0.3},
                {label: `평소 ${signed(r.usual)}`, order: 1, data: flat(r.usual), borderColor: color('--accent'),
                    borderWidth: 1, borderDash: [6, 4], pointRadius: 0},
                {label: '평소 범위', order: 2, data: flat(r.usual + r.spread), borderColor: 'transparent',
                    pointRadius: 0, backgroundColor: color('--line'), fill: '+1'},
                {label: '', order: 3, data: flat(r.usual - r.spread), borderColor: 'transparent', pointRadius: 0},
            ],
        },
        options: {
            animation: false,
            maintainAspectRatio: false,
            scales: {
                x: yearAxis(years, color),
                y: {ticks: {color: color('--muted'), callback: (v) => `${v}%`}, grid: {color: color('--line')}},
            },
            plugins: {
                legend: {labels: {color: color('--muted'), boxHeight: 2, filter: (item) => item.text !== ''}},
                tooltip: {
                    filter: (item) => item.datasetIndex === 0,
                    callbacks: {
                        label: (ctx) => `${r.points[ctx.dataIndex].month} 구 대비 ${signed(ctx.raw.y)}`,
                    },
                },
            },
        },
    }));
}

// --- buy or rent ---------------------------------------------------------------

const man = (v) => `${Math.round(v).toLocaleString()}만`;
const money = (v) => (Math.abs(v) >= 10000 ? eok(v) : man(v));

// buyRentSection is the buy-or-rent form for one unit type of a complex.
// Money is entered in 억, as people think of it, and sent to Go in 만원.
// Every change is recalculated and saved for next time.
function buyRentSection(c, area) {
    const result = h('div', {}, h('p', {class: 'sub'}, '불러오는 중…'));
    const section = h('div', {}, h('h2', {}, '매수 vs 전세'),
        h('p', {class: 'sub'}, '같은 현금으로 사는 것과 전세 사는 것의 1년 비용을 비교해서, 집값이 연 몇 % 올라야 사는 쪽이 이득인지 계산해요.'),
        result);

    BuyRentStart(c.sggCd, c.aptSeq, area).then((start) => {
        const inp = start.input;
        const field = (label, value, opts = {}) => {
            const input = h('input', {type: 'number', step: opts.step || 'any', min: 0, value: value ?? '',
                placeholder: opts.placeholder || ''});
            return {input, node: h('label', {}, label, input, opts.hint && h('span', {class: 'hint'}, opts.hint))};
        };
        const eokOf = (manwon) => (manwon ? +(manwon / 10000).toFixed(2) : '');
        const f = {
            price: field('매매가 (억)', eokOf(inp.price), {step: 0.01, hint: '기본은 최근 6개월 중간값'}),
            jeonse: field('전세가 (억)', eokOf(inp.jeonse), {step: 0.01, hint: '네이버 전세 호가 등'}),
            cash: field('보유 현금 (억)', eokOf(inp.cash), {step: 0.01, hint: '이 PC에만 저장돼요'}),
            loanRate: field('주담대 금리 (%)', inp.loanRate, {step: 0.1, hint: '26년 6월 은행 평균 4.36% · 고정 4.53%'}),
            jeonseRate: field('전세대출 금리 (%)', inp.jeonseRate, {step: 0.1, hint: '26년 6월 평균 4.07% · 현금이 모자랄 때만 쓰여요'}),            oppRate: field('기회수익률 (%)', inp.oppRate, {step: 0.1, hint: '예금 3% 안팎 · 국고채 3%대 · 주식 장기 6~8%'}),
            maintRate: field('유지보수 (%/년)', inp.maintRate, {step: 0.1, hint: '구축이면 높게'}),
            years: field('보유 기간 (년)', inp.years, {step: 1, hint: '취득 비용을 나눌 기간'}),
            tax: field('보유세 (만원/년)', inp.propertyTax || '', {step: 1, placeholder: '자동 추정',
                hint: '고지서 금액을 알면 넣으세요'}),
        };
        const out = h('div');
        const read = () => ({
            price: Math.round((+f.price.input.value || 0) * 10000),
            jeonse: Math.round((+f.jeonse.input.value || 0) * 10000),
            cash: Math.round((+f.cash.input.value || 0) * 10000),
            area: inp.area,
            loanRate: +f.loanRate.input.value || 0,
            jeonseRate: +f.jeonseRate.input.value || 0,
            oppRate: +f.oppRate.input.value || 0,
            maintRate: +f.maintRate.input.value || 0,
            years: Math.max(1, Math.round(+f.years.input.value || 1)),
            propertyTax: Math.round(+f.tax.input.value || 0),
        });
        let timer = null;
        const recalc = () => {
            clearTimeout(timer);
            timer = setTimeout(async () => {
                const v = read();
                if (!v.price || !v.jeonse) {
                    out.replaceChildren(h('p', {class: 'notice'}, '매매가와 전세가를 넣으면 계산돼요.'));
                    return;
                }
                try {
                    out.replaceChildren(renderBuyRent(await BuyRent(c.aptSeq, v), start.growth, v));
                } catch (err) {
                    out.replaceChildren(h('p', {class: 'error'}, errorText(err)));
                }
            }, 250);
        };
        const form = h('div', {class: 'form-grid', oninput: recalc}, Object.values(f).map((x) => x.node));
        result.replaceChildren(form, out);
        recalc();
    }).catch((err) => result.replaceChildren(h('p', {class: 'error'}, errorText(err))));
    return section;
}

// pastVerdict sets the break-even against the long and the recent rise.
// They can disagree, as when a complex doubled since 2016 but went nowhere
// since the 2021 peak; then it says so rather than pick one.
function pastVerdict(g, breakEven) {
    const longOK = g.all >= breakEven;
    const recentOK = g.recent5 >= breakEven;
    if (!g.recent5) return longOK ? '과거 흐름이 이어지면 사는 쪽이 유리해요.' : '과거 흐름보다 더 올라야 본전이에요.';
    if (longOK && recentOK) return '장기로도 최근 5년으로도, 이 흐름이면 사는 쪽이 유리해요.';
    if (longOK) return '장기 흐름이면 사는 쪽이 유리하지만, 최근 5년 흐름으로는 본전이 안 돼요.';
    if (recentOK) return '최근 5년 흐름이면 사는 쪽이 유리하지만, 장기 흐름으로는 본전이 안 돼요.';
    return '장기로도 최근 5년으로도, 과거 흐름보다 더 올라야 본전이에요.';
}

// renderBuyRent states the break-even, puts it against how fast this
// complex's prices have risen, and shows where both costs come from.
function renderBuyRent(r, g, v) {
    const pctText = (x) => `${x.toFixed(1)}%`;
    const headline = r.breakEven > 0
        ? h('p', {class: 'headline'}, '이 가격에 사면 집값이 ', h('strong', {}, `연 ${pctText(r.breakEven)}`),
            ' 이상 올라야 전세보다 이득이에요.')
        : h('p', {class: 'headline'}, '지금 조건이면 집값이 그대로여도 ', h('strong', {}, '사는 쪽이 연 ' +
            money(r.rentTotal - r.buyTotal)), ' 덜 들어요.');
    const past = g.since && h('p', {},
        `이 단지는 ${monthLabel(g.since.replace('-', ''))}부터 연 ${pctText(g.all)}`,
        g.recent5 ? `, 최근 5년은 연 ${pctText(g.recent5)} 올랐어요. ` : ' 올랐어요. ',
        h('strong', {}, pastVerdict(g, r.breakEven)));
    const lines = (title, items, total) => h('div', {class: 'table'}, h('table', {},
        h('thead', {}, h('tr', {}, h('th', {}, title), h('th', {class: 'num'}, '연간'))),
        h('tbody', {}, items.map((l) => h('tr', {}, h('td', {class: 'wrap'}, l.label), h('td', {class: 'num'}, man(l.amount)))),
            h('tr', {class: 'total'}, h('td', {}, '합계'), h('td', {class: 'num'}, man(total))))));
    const sens = h('div', {class: 'table'}, h('table', {},
        h('thead', {}, h('tr', {}, h('th', {}, '기회수익률'), r.sensitivity.map((x) => h('th', {class: 'num'}, `${x.oppRate}%`)))),
        h('tbody', {}, h('tr', {}, h('td', {}, '손익분기 상승률'),
            r.sensitivity.map((x) => h('td', {class: 'num'}, `연 ${pctText(x.breakEven)}`))))));
    return h('div', {},
        headline,
        past,
        h('p', {class: 'status'},
            `대출 ${money(r.loan)} 필요 · 취득세·중개수수료 ${money(r.acquireCost)} · ` +
            `보유세 ${money(r.propertyTax)}${r.taxEstimated ? ' (추정)' : ''}`),
        h('div', {class: 'two-col'}, lines('살 때', r.buy, r.buyTotal), lines('전세 살 때', r.rent, r.rentTotal)),
        h('p', {class: 'sub'}, '기회수익률에 따라 이렇게 달라져요. 결론이 뒤집히면 투자 성향에 따라 갈리는 가격이에요.'),
        sens,
        h('p', {class: 'fine'},
            '1주택 기준이에요. 취득세는 2025년 요율, 중개수수료는 법정 상한, 보유세는 공시가를 매매가의 70%로 보고 ' +
            '특례 없이 어림한 값이에요. 세제가 바뀌었을 수 있으니 큰 결정 전에는 확인하세요. ' +
            `집값 상승은 안 넣었어요. 위의 ${pctText(r.breakEven)}가 그 자리예요.`),
    );
}

// --- sync ------------------------------------------------------------------

// The sync bar sits under the header on every page while watched districts
// are fetched, and keeps the result until the next sync.
EventsOn('sync', (p) => {
    syncBar.hidden = false;
    syncBar.className = 'running';
    syncBar.textContent = p.label
        ? `받는 중 ${p.done}/${p.total} · ${p.label}`
        : `받는 중 ${p.done}/${p.total}`;
});

let syncing = null;

// sync fetches the watched districts. A second call while one runs waits for
// the same run instead of starting another.
function sync() {
    if (!syncing) {
        syncing = SyncWatched()
            .then((r) => {
                if (r.error) {
                    syncBar.className = 'failed';
                    syncBar.textContent = r.error;
                } else if (r.months === 0) {
                    syncBar.hidden = true;
                } else {
                    syncBar.className = 'done';
                    syncBar.textContent = r.added || r.newlyCanceled
                        ? `${r.months}개월 받음 · 새로 신고된 거래 ${r.added}건 · 새로 해제된 거래 ${r.newlyCanceled}건`
                        : `${r.months}개월 받음 · 달라진 거래 없음`;
                }
                return r;
            })
            .catch((err) => {
                syncBar.hidden = false;
                syncBar.className = 'failed';
                syncBar.textContent = errorText(err);
            })
            .finally(() => { syncing = null; });
    }
    return syncing;
}

// watchButton adds or removes a complex from the watchlist. Adding starts a
// sync so the complex's history is fetched.
function watchButton(c, watched, onChange) {
    const button = h('button', {class: watched ? 'secondary' : false},
        watched ? '★ 관심 단지에서 빼기' : '☆ 관심 단지에 추가');
    button.addEventListener('click', async () => {
        try {
            if (watched) await Unwatch(c.aptSeq);
            else await Watch(c.aptSeq, c.aptNm, c.sggCd, c.umdNm);
            const next = watchButton(c, !watched, onChange);
            button.replaceWith(next);
            if (!watched) sync().then(onChange);
        } catch (err) {
            button.replaceWith(h('span', {class: 'error'}, errorText(err)));
        }
    });
    return button;
}

// --- API key ---------------------------------------------------------------

function showSetup(canCancel) {
    const input = h('input', {class: 'wide', type: 'password', autocomplete: 'off', placeholder: '인증키'});
    const message = h('p');
    const save = h('button', {type: 'submit'}, '확인하고 저장');
    const form = h('form', {
        onsubmit: async (e) => {
            e.preventDefault();
            message.replaceChildren();
            try {
                await busy(save, '확인하는 중…', () => SaveAPIKey(input.value));
                showHome();
                sync().then(() => showHome());
            } catch (err) {
                message.replaceChildren(h('span', {class: 'error'}, errorText(err)));
            }
        },
    }, h('label', {}, 'API 키', input), save);

    show('setup',
        h('h1', {}, 'API 키 설정'),
        h('p', {class: 'sub'},
            '공공데이터포털 마이페이지의 ',
            h('strong', {}, '“국토교통부_아파트 매매 실거래가 상세 자료”'),
            ' 인증키를 넣어 주세요. Encoding, Decoding 어느 쪽이든 돼요. 이 PC에만 저장돼요.'),
        form,
        message,
        canCancel && h('button', {class: 'link', onclick: showHome}, '취소'),
    );
    input.focus();
}

// --- home: watchlist ---------------------------------------------------------

async function showHome() {
    const body = h('div', {}, h('p', {class: 'sub'}, '불러오는 중…'));
    const refresh = h('button', {class: 'secondary', onclick: () => sync().then(showHome)}, '지금 새로 받기');
    show('home', h('div', {class: 'title'}, h('h1', {}, '관심 단지'), refresh), body);
    try {
        const items = await Watchlist();
        if (items.length === 0) {
            body.replaceChildren(
                h('p', {}, '아직 관심 단지가 없어요.'),
                h('p', {class: 'sub'}, '“거래 찾기”에서 구와 월로 단지를 찾은 뒤, 단지 화면에서 관심 단지로 추가하세요. ' +
                    '추가하면 그 구의 지난 거래를 모두 받아와요.'),
            );
            return;
        }
        body.replaceChildren(h('div', {class: 'table'}, h('table', {},
            h('thead', {}, h('tr', {},
                h('th', {}, '단지'), h('th', {class: 'num'}, '저장된 거래'), h('th', {}, '최근 거래'))),
            h('tbody', {}, items.map((w) => h('tr', {},
                h('td', {class: 'wrap'},
                    h('a', {onclick: () => showHistory(w)}, w.aptNm),
                    h('span', {class: 'small'}, `${gu(w.sggCd)} ${w.umdNm}`)),
                h('td', {class: 'num'}, String(w.sales)),
                h('td', {}, w.latest
                    ? `${w.latest.date} · ${eok(w.latest.price)} · ${w.latest.floor}층`
                    : h('span', {class: 'sub'}, '받는 중이거나 거래 없음')),
            ))),
        )));
    } catch (err) {
        body.replaceChildren(h('p', {class: 'error'}, errorText(err)));
    }
}

// showHistory shows one unit type of a watched complex: its sales on a
// chart and in a table, newest first, and how much history is saved. With
// area 0 the type with the most sales is shown.
async function showHistory(c, area = 0) {
    const body = h('div', {}, h('p', {class: 'sub'}, '불러오는 중…'));
    show('home',
        h('p', {}, h('button', {class: 'link', onclick: showHome}, '← 관심 단지')),
        h('div', {class: 'title'}, h('h1', {}, c.aptNm), watchButton(c, true, () => showHistory(c, area))),
        body,
    );
    try {
        const [a, cov] = await Promise.all([ComplexAnalysis(c.sggCd, c.aptSeq, area), Coverage(c.sggCd)]);
        const complete = cov.fetched >= cov.total;
        const used = a.trades.filter((r) => !r.reason).length;
        const canvas = h('canvas');
        const premiumCanvas = h('canvas');
        // replaceChildren does not skip false like h does, so build in h.
        body.replaceChildren(h('div', {},
            h('p', {class: 'sub'}, `${gu(c.sggCd)} ${c.umdNm}`),
            h('p', {class: complete ? 'status' : 'notice'},
                `${monthLabel(cov.from)}부터 ${cov.total}개월 중 ${cov.fetched}개월 받음`,
                !complete && ' · 나머지는 받는 중이거나 중간에 멈췄어요. ',
                !complete && h('button', {class: 'link', onclick: () => sync().then(() => showHistory(c, area))}, '이어 받기')),
            a.types.length > 1 && h('div', {class: 'types'}, a.types.map((t) => h('button', {
                class: t.area === a.area ? 'type active' : 'type',
                onclick: () => showHistory(c, t.area),
            }, `${t.area}㎡ · ${t.sales}건`))),
            a.trades.length === 0 && h('p', {}, '저장된 거래가 아직 없어요.'),
            a.trades.length > 0 && h('div', {},
                h('p', {class: 'sub'},
                    `전용 ${a.area}㎡ · 분석에 쓰는 거래 ${used}건 · 제외 ${a.trades.length - used}건 (해제·직거래·1~2층)`),
                summaryCards(a),
                h('div', {class: 'chart'}, canvas),
                a.trend.some((p) => p.radius > trendRadius) && h('p', {class: 'status'},
                    '추세선의 점선 구간은 거래가 적어서 앞뒤 최대 6개월까지 넓혀 본 값이에요. ' +
                    '그만큼 완만하게 그려져서, 실제 오르내림은 더 컸을 수 있어요.'),
                a.relative.points.length > 0 && h('div', {},
                    h('h2', {}, '구 대비 프리미엄'),
                    h('p', {class: 'sub'},
                        `${gu(c.sggCd)}의 다른 단지 전용 ${a.relative.bandMin}~${a.relative.bandMax}㎡ 거래와 비교한 값이에요. ` +
                        '구 평균엔 신축과 구축이 섞여 있어서, 숫자 자체보다 이 단지의 평소 수준과 비교해서 보세요.'),
                    h('div', {class: 'chart small-chart'}, premiumCanvas)),
                buyRentSection(c, a.area),
                h('h2', {}, '거래 내역'),
                tradeTable(a.trades.slice().reverse())),
        ));
        clearCharts();
        if (a.trades.length) drawChart(canvas, a);
        if (a.relative.points.length) drawPremium(premiumCanvas, a.relative);
    } catch (err) {
        body.replaceChildren(h('p', {class: 'error'}, errorText(err)));
    }
}

// --- search: one month -------------------------------------------------------

let query = {sgg: '', month: ''};

async function showSearch() {
    if (!query.month) query.month = await DefaultMonth();
    const sgg = h('select', {required: true},
        h('option', {value: ''}, '구 선택'),
        districts.map((d) => h('option', {value: d.code, selected: d.code === query.sgg}, d.name)));
    const month = h('input', {inputmode: 'numeric', maxlength: 6, value: query.month, required: true});
    const go = h('button', {type: 'submit'}, '보기');
    const result = h('div');
    const form = h('form', {
        onsubmit: async (e) => {
            e.preventDefault();
            query = {sgg: sgg.value.trim(), month: month.value.trim()};
            await loadMonth(go, result);
        },
    }, h('label', {}, '구', sgg), h('label', {}, '계약 월', month), go);

    show('search', h('h1', {}, '거래 찾기'), form, result);
    if (query.sgg) await loadMonth(go, result);
    else sgg.focus();
}

// loadMonth shows the saved copy of the month, or with refresh set, fetches
// it again and says what changed.
async function loadMonth(button, result, refresh = false) {
    result.replaceChildren();
    try {
        const get = refresh ? RefreshMonth : Month;
        const s = await busy(button, refresh ? '새로 받는 중…' : '불러오는 중…', () => get(query.sgg, query.month));
        result.replaceChildren(renderMonth(s, () => loadMonth(button, result, true)));
    } catch (err) {
        result.replaceChildren(h('p', {class: 'error'}, errorText(err)));
    }
}

function renderMonth(s, refresh) {
    const head = h('p', {class: 'sub'},
        `${gu(query.sgg)} · ${monthLabel(query.month)} · 거래 ${s.total}건 · 해제 ${s.canceled}건 · 직거래 ${s.direct}건`);
    const status = h('p', {class: 'status'},
        `${s.fetchedAt}에 받은 데이터 · `,
        h('button', {class: 'link', onclick: refresh}, '새로 받기'));
    const notice = s.compared && h('p', {class: 'notice'},
        s.added || s.newlyCanceled
            ? `지난번 이후 새로 신고된 거래 ${s.added}건 · 새로 해제된 거래 ${s.newlyCanceled}건`
            : '지난번과 달라진 거래가 없어요');
    if (s.complexes.length === 0) {
        return h('div', {}, head, status, notice, h('p', {}, '이 달에는 신고된 거래가 없어요.'));
    }
    const rows = s.complexes.map((c) => h('tr', {},
        h('td', {class: 'wrap'},
            h('a', {onclick: () => showComplex({...c, sggCd: query.sgg})}, c.aptNm),
            h('span', {class: 'small'}, c.umdNm)),
        h('td', {class: 'num'}, String(c.buildYear)),
        h('td', {class: 'num'}, String(c.sales)),
    ));
    return h('div', {}, head, status, notice, h('div', {class: 'table'}, h('table', {},
        h('thead', {}, h('tr', {}, h('th', {}, '단지'), h('th', {class: 'num'}, '준공'), h('th', {class: 'num'}, '거래'))),
        h('tbody', {}, rows),
    )));
}

// showComplex lists one complex's sales in the searched month.
async function showComplex(c) {
    const body = h('div', {}, h('p', {class: 'sub'}, '불러오는 중…'));
    const title = h('div', {class: 'title'}, h('h1', {}, c.aptNm));
    show('search',
        h('p', {}, h('button', {class: 'link', onclick: showSearch}, `← ${gu(query.sgg)} · ${monthLabel(query.month)}`)),
        title,
        body,
    );
    try {
        const [trades, watched] = await Promise.all([
            ComplexTrades(query.sgg, query.month, c.aptSeq),
            IsWatched(c.aptSeq),
        ]);
        title.append(watchButton(c, watched, () => {}));
        body.replaceChildren(
            h('p', {class: 'sub'}, `${gu(query.sgg)} ${c.umdNm} · ${monthLabel(query.month)} 거래 ${trades.length}건`),
            tradeTable(asRows(trades)),
        );
    } catch (err) {
        body.replaceChildren(h('p', {class: 'error'}, errorText(err)));
    }
}

// --- start -------------------------------------------------------------------

const pages = {home: showHome, search: showSearch, setup: () => showSetup(true)};
nav.addEventListener('click', (e) => {
    const page = e.target.dataset && e.target.dataset.page;
    if (page) pages[page]();
});

Promise.all([HasAPIKey(), Districts()])
    .then(([has, list]) => {
        districts = list;
        if (!has) return showSetup(false);
        showHome();
        // Opening the app is when it catches up: the last few months again,
        // and any month never fetched.
        sync().then(() => showHome());
    })
    .catch((err) => show('home', h('p', {class: 'error'}, errorText(err))));