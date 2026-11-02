import Chart from 'chart.js/auto';
import {
    ComplexAnalysis, ComplexTrades, Coverage, DefaultMonth, Districts, HasAPIKey, IsWatched, Month,
    RefreshMonth, SaveAPIKey, SyncWatched, Unwatch, Watch, Watchlist,
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

let chart = null;

const monthMs = (month) => Date.UTC(Number(month.slice(0, 4)), Number(month.slice(5, 7)) - 1, 15);
const markLabel = (m) => `${eok(m.price)} (${m.month.replace('-', '.')})`;
const signed = (v) => `${v > 0 ? '+' : ''}${v.toFixed(1)}%`;

// drawChart plots one unit type's sales over time: sales the analysis uses
// in the accent colour, direct and low-floor sales in grey, the rolling
// median as a line, and the peak, trough and recent median across. Canceled
// sales are left off; most are re-reported and would show twice.
function drawChart(canvas, a) {
    const rows = a.trades;
    const s = a.summary;
    const css = getComputedStyle(document.documentElement);
    const color = (name) => css.getPropertyValue(name).trim();
    const point = (r) => ({x: Date.parse(r.trade.date), y: r.trade.price / 10000, row: r});
    const used = rows.filter((r) => !r.reason).map(point);
    const left = rows.filter((r) => r.reason && r.reason !== 'canceled').map(point);
    const all = used.concat(left);
    const years = [];
    if (all.length) {
        const first = new Date(Math.min(...all.map((p) => p.x))).getUTCFullYear();
        const last = new Date(Math.max(...all.map((p) => p.x))).getUTCFullYear();
        for (let y = first; y <= last + 1; y++) years.push(Date.UTC(y, 0, 1));
    }
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
            data: a.trend.map((p) => ({x: monthMs(p.month), y: p.median ? p.median / 10000 : null, trend: p})),
        },
    ];
    if (s.peak) datasets.push(across(`고점 ${markLabel(s.peak)}`, s.peak.price, color('--muted')));
    if (s.trough) datasets.push(across(`바닥 ${markLabel(s.trough)}`, s.trough.price, color('--warn')));
    if (s.recentMedian) datasets.push(across(`최근 ${eok(s.recentMedian)}`, s.recentMedian, color('--accent')));

    if (chart) chart.destroy();
    chart = new Chart(canvas, {
        type: 'scatter',
        data: {datasets},
        options: {
            animation: false,
            maintainAspectRatio: false,
            scales: {
                x: {
                    type: 'linear', min: years[0], max: years[years.length - 1],
                    afterBuildTicks: (axis) => { axis.ticks = years.map((value) => ({value})); },
                    ticks: {color: color('--muted'), callback: (v) => new Date(v).getUTCFullYear()},
                    grid: {color: color('--line')},
                },
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
                            if (raw.trend) return `${raw.trend.month} 추세 ${eok(raw.trend.median)} (${raw.trend.sales}건)`;
                            if (!raw.row) return ctx.dataset.label;
                            const {trade: t, reason} = raw.row;
                            const why = reason ? ` · ${reasonLabels[reason]}` : '';
                            return `${t.date} · ${eok(t.price)} · ${t.floor}층${why}`;
                        },
                    },
                },
            },
        },
    });
}

// summaryCards leads the complex screen: where recent prices sit against
// the peak and the trough of the trend line.
function summaryCards(s) {
    const card = (label, value, note) => h('div', {class: 'card'},
        h('span', {class: 'small'}, label), h('strong', {}, value), note && h('span', {class: 'small'}, note));
    return h('div', {class: 'cards'},
        card('최근 6개월 중간값', s.recentMedian ? eok(s.recentMedian) : '거래 없음',
            `${s.recentSales}건 · ${s.recentFrom} 이후`),
        card('고점', s.peak ? eok(s.peak.price) : '—', s.peak && monthLabel(s.peak.month.replace('-', ''))),
        card('하락기 바닥', s.trough ? eok(s.trough.price) : '—', s.trough && monthLabel(s.trough.month.replace('-', ''))),
        card('지금 위치',
            s.recentMedian && s.trough ? `바닥 ${signed(s.fromTrough)}` : '—',
            s.recentMedian && s.peak ? `고점 ${signed(s.fromPeak)}` : null),
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
                summaryCards(a.summary),
                h('div', {class: 'chart'}, canvas),
                tradeTable(a.trades.slice().reverse())),
        ));
        if (a.trades.length) drawChart(canvas, a);
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