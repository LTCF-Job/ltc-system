import assert from 'node:assert/strict'
import test from 'node:test'
import { getTripPatternDisplay } from '../../src/lib/rideCalendarDisplay.ts'

test('does not show zero trips for a case without a schedule', () => {
  assert.equal(
    getTripPatternDisplay({ tripPattern: 0, days: {} }),
    '未設定排班'
  )
})

test('uses expected daily trip counts when a schedule exists', () => {
  assert.equal(
    getTripPatternDisplay({
      tripPattern: 0,
      days: {
        '2026-07-01': {
          isExpected: true,
          expectedTripCount: 2
        }
      }
    }),
    '2 趟'
  )
})
