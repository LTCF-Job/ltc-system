import { casesHandlers } from './cases'
import { mastersHandlers } from './masters'
import { formsHandlers } from './forms'
import { ridesHandlers } from './rides'
import { exportsHandlers } from './exports'
import { reportsHandlers } from './reports'
import { operationsHandlers } from './operations'
import { systemHandlers } from './system'

export const handlers = [
  ...casesHandlers,
  ...mastersHandlers,
  ...formsHandlers,
  ...ridesHandlers,
  ...exportsHandlers,
  ...reportsHandlers,
  ...operationsHandlers,
  ...systemHandlers
]
