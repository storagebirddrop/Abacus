import { afterEach, describe, expect, it, vi, type Mock } from 'vitest'

vi.mock('./client', () => ({ apiFetch: vi.fn().mockResolvedValue({}) }))

import { apiFetch } from './client'
import { startSync, getSyncJob, listSyncJobs } from './sync'

const mock = apiFetch as unknown as Mock

afterEach(() => mock.mockClear())

describe('sync API contract', () => {
  it('startSync → POST /wallets/{id}/sync', () => {
    startSync('w1')
    expect(mock).toHaveBeenCalledWith('/wallets/w1/sync', { method: 'POST' })
  })

  it('getSyncJob → GET /sync-jobs/{id}', () => {
    getSyncJob('job1')
    expect(mock).toHaveBeenCalledWith('/sync-jobs/job1')
  })

  it('listSyncJobs → GET /wallets/{id}/sync-jobs', () => {
    listSyncJobs('w1')
    expect(mock).toHaveBeenCalledWith('/wallets/w1/sync-jobs')
  })
})
