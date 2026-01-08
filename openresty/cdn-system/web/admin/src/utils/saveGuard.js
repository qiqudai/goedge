const normalizeValue = (value) => {
  if (value === undefined || value === null) {
    return ''
  }
  return String(value)
}

const isRequiredField = (el) => {
  if (el.required || el.hasAttribute('required')) {
    return true
  }
  if (el.getAttribute('aria-required') === 'true') {
    return true
  }
  const formItem = el.closest('.el-form-item')
  return !!formItem && formItem.classList.contains('is-required')
}

export const cacheInputValue = (event) => {
  const el = event?.target
  if (!(el instanceof HTMLInputElement || el instanceof HTMLTextAreaElement)) {
    return
  }
  el.dataset.lastValue = normalizeValue(el.value)
}

export const shouldSkipBlurSave = (event) => {
  const el = event?.target
  if (!(el instanceof HTMLInputElement || el instanceof HTMLTextAreaElement)) {
    return false
  }
  const value = normalizeValue(el.value)
  const lastValue = normalizeValue(el.dataset.lastValue)
  if (value === lastValue) {
    return true
  }
  if (value === '' && isRequiredField(el)) {
    return true
  }
  el.dataset.lastValue = value
  return false
}
