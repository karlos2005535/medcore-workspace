import grpc from "k6/net/grpc";
import { check } from "k6";

const client = new grpc.Client();
client.load(["../../proto"], "pharmacy.proto");

// Connect sekali di init context (shared antar VU)
client.connect("localhost:50051", { plaintext: true });

export const options = {
  stages: [
    { duration: "10s", target: 50 },
    { duration: "20s", target: 200 },
    { duration: "10s", target: 0 },
  ],
};

export default function () {
  const response = client.invoke(
    "medcore.pharmacy.v1.PharmacyService/CheckDrugAvailability",
    { drug_code: "MED-AMX-500", quantity_needed: 10 },
  );

  check(response, {
    "status is OK": (r) => r && r.status === grpc.StatusOK,
    "has valid stock": (r) => r && r.message.isAvailable === true,
  });
}
