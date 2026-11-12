import assert from "node:assert/strict";
import { test } from "node:test";
import { hasValidEntityIds, isEntityId, parseEntityId } from "./entity-id.ts";

test("accepts canonical positive decimal entity IDs without number conversion", () => {
  assert.equal(parseEntityId("1"), "1");
  assert.equal(parseEntityId("42"), "42");
  assert.equal(parseEntityId("9007199254740993"), "9007199254740993");
  assert.equal(parseEntityId("9223372036854775807"), "9223372036854775807");
});

test("rejects values outside the positive signed bigint range", () => {
  for (const value of ["", "0", "01", "-1", "+1", " 1", "1 ", "1.0", "abc", "9223372036854775808"]) {
    assert.equal(parseEntityId(value), null, "accepted value " + JSON.stringify(value));
  }
  assert.equal(parseEntityId(42), null);
  assert.equal(isEntityId("3db8b61c-39a3-4ce9-b66a-0d7a9ed9b147"), false);
});

test("validates entity IDs recursively but leaves UUID command IDs alone", () => {
  assert.equal(hasValidEntityIds({
    order: { id: "12", eventId: "9007199254740993" },
    clientOrderId: "3db8b61c-39a3-4ce9-b66a-0d7a9ed9b147",
  }), true);
  assert.equal(hasValidEntityIds({ items: [{ id: "42" }, { bundleListingId: "bad-id" }] }), false);
});
