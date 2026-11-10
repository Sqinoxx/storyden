import { test } from "uvu";
import * as assert from "uvu/assert";

import {
  dateInputValue,
  endOfDayFromInput,
  expiryPresets,
  termEnd,
} from "./expiry";

test("winter ends on 31 March of the following year", () => {
  const end = termEnd({ year: 2026, winter: true });
  assert.is(dateInputValue(end), "2027-03-31");
});

test("summer ends on 30 September", () => {
  const end = termEnd({ year: 2026, winter: false });
  assert.is(dateInputValue(end), "2026-09-30");
});

test("presets during the winter term", () => {
  const [none, thisTerm, nextTerm] = expiryPresets(new Date(2026, 10, 4));
  assert.is(none?.key, "none");
  assert.is(none?.expiresAt, undefined);
  assert.is(thisTerm?.label, "WS26/27");
  assert.is(dateInputValue(thisTerm!.expiresAt!), "2027-03-31");
  assert.is(nextTerm?.label, "SS27");
  assert.is(dateInputValue(nextTerm!.expiresAt!), "2027-09-30");
});

test("presets in January still belong to the winter term", () => {
  const [, thisTerm, nextTerm] = expiryPresets(new Date(2027, 0, 15));
  assert.is(thisTerm?.label, "WS26/27");
  assert.is(nextTerm?.label, "SS27");
});

test("presets during the summer term", () => {
  const [, thisTerm, nextTerm] = expiryPresets(new Date(2026, 5, 1));
  assert.is(thisTerm?.label, "SS26");
  assert.is(nextTerm?.label, "WS26/27");
  assert.is(dateInputValue(nextTerm!.expiresAt!), "2027-03-31");
});

test("date input parsing", () => {
  const d = endOfDayFromInput("2027-02-28");
  assert.ok(d);
  assert.is(d!.getHours(), 23);
  assert.is(dateInputValue(d!), "2027-02-28");
  assert.is(endOfDayFromInput("28.02.2027"), undefined);
  assert.is(endOfDayFromInput(""), undefined);
});

test.run();
