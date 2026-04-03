import http from "k6/http";
import { check, sleep } from "k6";

export let options = {
  vus: 50,
  duration: "60s",
  thresholds: {
    http_req_duration: ["p(95)<250"],
  },
};

export default function () {
  // Nantinya localhost:8080 bisa lo ganti jadi IP VPS lo di .env
  const url = "http://localhost:8080/mocks/v1/payment/check";

  const payload = "";
  const params = {
    headers: { "X-Account-Type": "premium" },
  };

  let res;
  // Logika otomatis pilih Method sesuai database
  if ("POST" === "POST") {
    res = http.post(url, payload, params);
  } else {
    res = http.get(url, params);
  }

  check(res, {
    "status is 200": (r) => r.status === 200,
  });

  sleep(1);
}
