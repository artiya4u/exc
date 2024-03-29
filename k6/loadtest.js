import http from 'k6/http'
import {check} from 'k6';

export let options = {
    thresholds: {
        http_req_duration: ['p(95)<100', 'p(99)<500'],
    },
    stages: [
        {duration: '1m', target: 5000},
    ],
}

export default function () {
    const responses = http.batch([
        {
            method: 'POST',
            url: 'http://localhost:8000/shorten',
            body: JSON.stringify({
                "url": "https://developer.mozilla.org/en-US/docs/Web/HTTP/Status/302"
            }),
            params: {
                headers: {'Content-Type': 'application/json'},
            }
        },
        {
            method: 'GET',
            url: 'http://localhost:8000/shorten/https://developer.mozilla.org/en-US/docs/Web/HTTP/Status/302',
        },
        {
            method: 'GET',
            url: 'http://localhost:8000/uivKV6',
            params: {
                redirects: 0
            }
        }
    ]);
    check(responses[0], {
        'shorten post success': (res) => res.status === 200,
    });
    check(responses[1], {
        'shorten get success': (res) => res.status === 200,
    });
    check(responses[2], {
        'redirect success': (res) => res.status === 302,
    });
}