import http from "k6/http";
import { check } from "k6";

export const options = {
  stages: [
    { duration: "10s", target: 50 }, // ramp up 50 VUs
    { duration: "20s", target: 200 }, // spike ke 200 VUs
    { duration: "10s", target: 0 }, // cool down
  ],
  summaryTrendStats: ["avg", "min", "med", "p(90)", "p(95)", "p(99)", "max"],
};

export default function () {
  const res = http.get(
    "http://localhost:8081/api/v1/drugs/check?drug_code=MED-AMX-500",
  );
  check(res, {
    "status is 200": (r) => r.status === 200,
    "has valid json": (r) => r.json().drug_code === "MED-AMX-500",
  });
}
