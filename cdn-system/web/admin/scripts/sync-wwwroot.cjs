#!/usr/bin/env node
const fs = require('fs')
const path = require('path')
const { spawnSync } = require('child_process')

const rootDir = path.resolve(__dirname, '..')
const distDir = path.join(rootDir, 'dist')
const targetDir = '/www/wwwroot/www'
const watchMode = process.argv.includes('--watch')

const ensureDir = (dir) => {
  fs.mkdirSync(dir, { recursive: true })
}

const runCopy = () => {
  if (!fs.existsSync(distDir)) {
    console.error('dist not found; run npm run build first')
    return false
  }
  ensureDir(targetDir)
  const result = spawnSync('cp', ['-a', `${distDir}/.`, `${targetDir}/`], { stdio: 'inherit' })
  return result.status === 0
}

const getLatestMtime = (dir) => {
  let latest = 0
  const entries = fs.readdirSync(dir, { withFileTypes: true })
  for (const entry of entries) {
    const fullPath = path.join(dir, entry.name)
    if (entry.isDirectory()) {
      const child = getLatestMtime(fullPath)
      if (child > latest) latest = child
    } else if (entry.isFile()) {
      const stat = fs.statSync(fullPath)
      const mtime = stat.mtimeMs || 0
      if (mtime > latest) latest = mtime
    }
  }
  return latest
}

const sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms))

const runWatch = async () => {
  let lastMtime = 0
  let ready = false
  while (true) {
    if (fs.existsSync(distDir)) {
      const mtime = getLatestMtime(distDir)
      if (!ready || mtime > lastMtime) {
        lastMtime = mtime
        ready = true
        runCopy()
      }
    }
    await sleep(1000)
  }
}

if (watchMode) {
  runWatch()
} else {
  runCopy()
}
