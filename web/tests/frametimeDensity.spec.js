import { Buffer } from 'node:buffer'
import { test, expect } from '@playwright/test'

test('frametime density preserves decimal buckets from upload to chart tooltip', async ({ page, request }, testInfo) => {
  const login = await request.post('/auth/admin/login', {
    data: { username: 'e2eadmin', password: 'e2epassword1' }
  })
  expect(login.ok()).toBeTruthy()

  const values = [0.1, 99, ...Array.from({ length: 50 }, () => [3.14, 3.24, 3.34]).flat()]
  const csv = [
    'os,cpu,gpu,ram,kernel,driver,cpuscheduler',
    'Linux,Test CPU,Test GPU,16000000,test,test,test',
    'fps,frametime',
    ...values.map(value => `${1000 / value},${value}`)
  ].join('\n')
  const upload = await request.post('/api/benchmarks', {
    multipart: {
      title: 'Frametime density precision regression',
      files: { name: 'precision.csv', mimeType: 'text/csv', buffer: Buffer.from(csv) }
    }
  })
  expect(upload.ok()).toBeTruthy()
  const benchmark = await upload.json()
  try {
    const response = await request.get(`/api/benchmarks/${benchmark.id}/runs/0`)
    expect(response.ok()).toBeTruthy()
    const run = await response.json()
    for (const method of ['stats', 'statsMangoHud']) {
      expect(run[method].FrameTime.density).toEqual([[3.1, 50], [3.2, 50], [3.3, 50]])
      expect(run[method].FPS.density.every(([value]) => Number.isInteger(value))).toBeTruthy()
    }
    const debugResponse = await request.post('/api/debugcalc', { data: { frameTime: values } })
    expect(debugResponse.ok()).toBeTruthy()
    const debug = await debugResponse.json()
    expect(debug.linear.frameTime.density).toEqual(run.stats.FrameTime.density)
    expect(debug.mangohud.frameTime.density).toEqual(run.statsMangoHud.FrameTime.density)

    await page.goto(`/benchmarks/${benchmark.id}`)
    await page.locator('#frametime-tab').click()
    const chart = page.locator('.highcharts-container').filter({ has: page.getByText('Frametime Density', { exact: true }) })
    await expect(chart).toBeVisible()
    await chart.scrollIntoViewIfNeeded()
    const graph = chart.locator('.highcharts-graph').first()
    await expect(graph).toBeAttached()
    const box = await graph.boundingBox()
    await page.mouse.move(box.x + box.width / 2, box.y + Math.max(box.height / 2, 1), { steps: 10 })
    await page.mouse.move(box.x + box.width / 2 + 1, box.y + Math.max(box.height / 2, 1))
    await expect(chart.locator('.highcharts-tooltip')).toContainText('50 points at ~3.2 ms')
    await chart.screenshot({ path: testInfo.outputPath('frametime-density.png') })
  } finally {
    const removed = await request.delete(`/api/benchmarks/${benchmark.id}`)
    expect(removed.ok()).toBeTruthy()
  }
})
