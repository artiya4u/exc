import http from 'k6/http'
import {check} from 'k6';

export let options = {
    thresholds: {
        http_req_duration: ['p(95)<100', 'p(99)<500'],
    },
    stages: [
        {duration: '1m', target: 1000},
    ],
}

export default function () {
    const url = 'http://localhost:8000/shorten'
    const data = {
        "url": "https://mirror.xyz/myalphadrops.eth/o3NTGGOJFkoqR7IzFcWbTmL5-mLwUArYQ_K0XpwC88M"
    }
    let res = http.post(url, JSON.stringify(data), {
        headers: {'Content-Type': 'application/json'},
    });

    check(res, {'success': (r) => r.status === 200})

    const responses = http.batch([
        {
            method: 'POST',
            url: 'https://httpbin.test.k6.io/post',
            body: {
                "url": "https://mirror.xyz/myalphadrops.eth/o3NTGGOJFkoqR7IzFcWbTmL5-mLwUArYQ_K0XpwC88M"
            },
            params: {
                headers: {'Content-Type': 'application/json'},
            },
        },
        {
            method: 'GET',
            url: 'http://localhost:8000/6iDv7Ckn2oQ',
            params: {
                redirects: 0
            }
        }
    ]);
    check(responses[0], {
        'shorten success': (res) => res.status === 200,
    });
    check(responses[1], {
        'redirect success': (res) => res.status === 307,
    });
}