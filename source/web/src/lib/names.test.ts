import { describe, expect, it } from "vitest";
import { liveFirstName, liveLastName, normalizeFirstName, normalizeLastName } from "./names";

describe("normalizeLastName", () => {
  it.each([
    ["anderson", "ANDERSON"],
    ["  van   der berg ", "VAN DER BERG"],
    ["Åkesson", "ÅKESSON"],
    ["øye", "ØYE"],
    ["o'brien", "O'BRIEN"],
  ])("%j -> %j", (input, want) => expect(normalizeLastName(input)).toBe(want));
});

describe("normalizeFirstName", () => {
  it.each([
    ["mIKE", "Mike"],
    ["anna-lise", "Anna-Lise"],
    ["o'neil", "O'Neil"],
    ["d’arcy", "D’Arcy"],
    ["  mary   ann ", "Mary Ann"],
    ["åse bjørg", "Åse Bjørg"],
    ["ÉMILIE", "Émilie"],
    ["ßa", "ßa"],
  ])("%j -> %j", (input, want) => expect(normalizeFirstName(input)).toBe(want));
});

describe("live variants", () => {
  it("keep whitespace so the caret stays usable", () => {
    expect(liveFirstName("anna  ")).toBe("Anna  ");
    expect(liveFirstName("anna- l")).toBe("Anna- L");
    expect(liveLastName(" van ")).toBe(" VAN ");
  });
});
