import {ComplexTrades, DefaultMonth, HasAPIKey, Month, SaveAPIKey} from '../wailsjs/go/main/App';

const app = document.getElementById('app');
const settingsButton = document.getElementById('settings');

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
function show(...nodes) {
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

// --- API key ---------------------------------------------------------------

function showSetup(canCancel) {
    settingsButton.hidden = true;
    const input = h('input', {class: 'wide', type: 'password', autocomplete: 'off', placeholder: 'Decoding 키'});
    const message = h('p');
    const save = h('button', {type: 'submit'}, '확인하고 저장');
    const form = h('form', {
        onsubmit: async (e) => {
            e.preventDefault();
            message.replaceChildren();
            try {
                await busy(save, '확인하는 중…', () => SaveAPIKey(input.value));
                showSearch();
            } catch (err) {
                message.replaceChildren(h('span', {class: 'error'}, errorText(err)));
            }
        },
    }, h('label', {}, 'API 키', input), save);

    show(
        h('h1', {}, 'API 키 설정'),
        h('p', {class: 'sub'},
            '공공데이터포털 마이페이지의 ',
            h('strong', {}, '“국토교통부_아파트 매매 실거래가 상세 자료”'),
            ' Decoding 키를 넣어 주세요. 이 PC에만 저장돼요.'),
        form,
        message,
        canCancel && h('button', {class: 'link', onclick: showSearch}, '취소'),
    );
    input.focus();
}

// --- month -----------------------------------------------------------------

let query = {sgg: '', month: ''};

async function showSearch() {
    settingsButton.hidden = false;
    if (!query.month) query.month = await DefaultMonth();
    const sgg = h('input', {inputmode: 'numeric', maxlength: 5, placeholder: '11110', value: query.sgg, required: true});
    const month = h('input', {inputmode: 'numeric', maxlength: 6, value: query.month, required: true});
    const go = h('button', {type: 'submit'}, '보기');
    const result = h('div');
    const form = h('form', {
        onsubmit: async (e) => {
            e.preventDefault();
            query = {sgg: sgg.value.trim(), month: month.value.trim()};
            await loadMonth(go, result);
        },
    }, h('label', {}, '구 코드', sgg), h('label', {}, '계약 월', month), go);

    show(h('h1', {}, '한 달 거래 보기'), form, result);
    if (query.sgg) await loadMonth(go, result);
    else sgg.focus();
}

async function loadMonth(button, result) {
    result.replaceChildren();
    try {
        const s = await busy(button, '불러오는 중…', () => Month(query.sgg, query.month));
        result.replaceChildren(renderMonth(s));
    } catch (err) {
        result.replaceChildren(h('p', {class: 'error'}, errorText(err)));
    }
}

function renderMonth(s) {
    const head = h('p', {class: 'sub'},
        `${query.sgg} · ${monthLabel(query.month)} · 거래 ${s.total}건 · 해제 ${s.canceled}건 · 직거래 ${s.direct}건`);
    if (s.complexes.length === 0) return h('div', {}, head, h('p', {}, '이 달에는 신고된 거래가 없어요.'));
    const rows = s.complexes.map((c) => h('tr', {},
        h('td', {class: 'wrap'},
            h('a', {onclick: () => showComplex(c.aptSeq, c.aptNm)}, c.aptNm),
            h('span', {class: 'small'}, c.umdNm)),
        h('td', {class: 'num'}, String(c.buildYear)),
        h('td', {class: 'num'}, String(c.sales)),
    ));
    return h('div', {}, head, h('div', {class: 'table'}, h('table', {},
        h('thead', {}, h('tr', {}, h('th', {}, '단지'), h('th', {class: 'num'}, '준공'), h('th', {class: 'num'}, '거래'))),
        h('tbody', {}, rows),
    )));
}

// --- complex ---------------------------------------------------------------

async function showComplex(aptSeq, aptNm) {
    const body = h('div', {}, h('p', {class: 'sub'}, '불러오는 중…'));
    show(
        h('p', {}, h('button', {class: 'link', onclick: showSearch}, `← ${query.sgg} · ${monthLabel(query.month)}`)),
        h('h1', {}, aptNm),
        body,
    );
    try {
        const trades = await ComplexTrades(query.sgg, query.month, aptSeq);
        body.replaceChildren(renderTrades(aptSeq, trades));
    } catch (err) {
        body.replaceChildren(h('p', {class: 'error'}, errorText(err)));
    }
}

function renderTrades(aptSeq, trades) {
    const rows = trades.map((t) => h('tr', {class: t.canceled || t.direct ? 'dim' : false},
        h('td', {}, t.date),
        h('td', {class: 'num'}, t.area.toFixed(2)),
        h('td', {class: 'num'}, String(t.floor)),
        h('td', {class: 'num'}, eok(t.price)),
        h('td', {class: 'wrap'},
            t.canceled && h('span', {class: 'flag warn', title: `해제일 ${t.canceledOn || ''}`},
                t.canceledOn ? `해제 ${t.canceledOn.slice(5)}` : '해제'),
            t.direct && h('span', {class: 'flag warn'}, '직거래'),
            !t.canceled && !t.registeredOn && h('span', {class: 'flag'}, '미등기')),
    ));
    return h('div', {},
        h('p', {class: 'sub'}, `${aptSeq} · ${monthLabel(query.month)} 거래 ${trades.length}건`),
        h('div', {class: 'table'}, h('table', {},
            h('thead', {}, h('tr', {},
                h('th', {}, '계약일'), h('th', {class: 'num'}, '전용㎡'), h('th', {class: 'num'}, '층'),
                h('th', {class: 'num'}, '금액'), h('th', {}, '표시'))),
            h('tbody', {}, rows),
        )));
}

// --- start -----------------------------------------------------------------

settingsButton.addEventListener('click', () => showSetup(true));

HasAPIKey()
    .then((has) => (has ? showSearch() : showSetup(false)))
    .catch((err) => show(h('p', {class: 'error'}, errorText(err))));