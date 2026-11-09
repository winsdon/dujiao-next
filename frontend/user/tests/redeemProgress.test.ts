import test from 'node:test'
import assert from 'node:assert/strict'

import {
  formatElapsed,
  parseCstDateTime,
  resolveRedeemPhase,
  resolveTimelineStates,
} from '../src/utils/redeemProgress.ts'

test('captcha takes priority over status when resolving phase', () => {
  assert.equal(resolveRedeemPhase('processing', true), 'captcha')
  assert.equal(resolveRedeemPhase('pending_confirm', true), 'captcha')
})

test('pending_confirm maps to confirming, others to processing', () => {
  assert.equal(resolveRedeemPhase('pending_confirm', false), 'confirming')
  assert.equal(resolveRedeemPhase('processing', false), 'processing')
  assert.equal(resolveRedeemPhase('locked', false), 'processing')
  assert.equal(resolveRedeemPhase(undefined, false), 'processing')
})

test('timeline states follow phase', () => {
  assert.deepEqual(resolveTimelineStates('processing'), {
    submitted: 'done',
    processing: 'active',
    captcha: 'idle',
    confirm: 'idle',
  })
  assert.deepEqual(resolveTimelineStates('captcha'), {
    submitted: 'done',
    processing: 'paused',
    captcha: 'attention',
    confirm: 'idle',
  })
  assert.deepEqual(resolveTimelineStates('confirming'), {
    submitted: 'done',
    processing: 'done',
    captcha: 'idle',
    confirm: 'active',
  })
})

test('parseCstDateTime treats backend time as UTC+8', () => {
  assert.equal(parseCstDateTime('2026-01-02 08:00:00'), Date.UTC(2026, 0, 2, 0, 0, 0))
  assert.equal(parseCstDateTime('2026-01-02T08:00:00'), Date.UTC(2026, 0, 2, 0, 0, 0))
})

test('parseCstDateTime rejects empty or malformed values', () => {
  assert.equal(parseCstDateTime(''), null)
  assert.equal(parseCstDateTime(undefined), null)
  assert.equal(parseCstDateTime('not a date'), null)
  assert.equal(parseCstDateTime('2026-13-40 99:99:99'), null)
})

test('formatElapsed renders mm:ss and h:mm:ss', () => {
  assert.equal(formatElapsed(0), '00:00')
  assert.equal(formatElapsed(83), '01:23')
  assert.equal(formatElapsed(3600 + 62), '1:01:02')
  assert.equal(formatElapsed(-5), '00:00')
  assert.equal(formatElapsed(9.9), '00:09')
})
