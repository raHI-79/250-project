import assert from 'node:assert/strict'
import test from 'node:test'
import {
  batchClosedText,
  batchFromRegNo,
  batchNameProblem,
  cleanBatchName,
  regNoProblem,
} from './batches.js'

test('the batch comes from the first four digits of the registration number', () => {
  assert.equal(batchFromRegNo('2021331001'), 'CSE-21')
  assert.equal(batchFromRegNo('2023331079'), 'CSE-23')
  assert.equal(batchFromRegNo('202533100112'), 'CSE-25')
  assert.equal(batchFromRegNo(' 2025'), 'CSE-25')
})

test('no batch is shown until the registration number names one', () => {
  assert.equal(batchFromRegNo(''), '')
  assert.equal(batchFromRegNo('202'), '')
  assert.equal(batchFromRegNo('1923331079'), '')
})

test('registration numbers are digits that start with 20 and have 6 to 20 digits', () => {
  assert.equal(regNoProblem('2023331079'), '')
  assert.equal(regNoProblem('202333107912'), '')
  assert.match(regNoProblem(''), /required/)
  assert.match(regNoProblem('2023A31079'), /digits/)
  assert.match(regNoProblem('20233'), /6 to 20/)
  assert.match(regNoProblem('3023331079'), /start with 20/)
})

test('a closed batch gets the office message', () => {
  assert.equal(
    batchClosedText('CSE-25'),
    'CSE-25 is not open for registration. Please contact the department office.',
  )
})

test('a batch name is upper-cased and has to look like CSE-25', () => {
  assert.equal(cleanBatchName(' cse-25 '), 'CSE-25')
  assert.equal(batchNameProblem('cse-25'), '')
  assert.match(batchNameProblem('25'), /CSE-25/)
})
