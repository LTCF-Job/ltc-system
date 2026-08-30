import { casesHandlers } from './cases'
import { mastersHandlers } from './masters'
import { caregiversHandlers } from './caregivers'
import { formsHandlers } from './forms'
import { ridesHandlers } from './rides'
import { exportsHandlers } from './exports'
import { reportsHandlers } from './reports'
import { operationsHandlers } from './operations'
import { holidaysHandlers } from './holidays'
import { systemHandlers } from './system'

export const handlers = [
  ...casesHandlers,
  ...mastersHandlers,
  ...caregiversHandlers,
  ...formsHandlers,
  ...ridesHandlers,
  ...exportsHandlers,
  ...reportsHandlers,
  ...operationsHandlers,
  ...holidaysHandlers,
  ...systemHandlers
]
