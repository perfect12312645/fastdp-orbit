/** 校验工具函数集合 */

/** 校验IP地址 */
export function validateIP(_rule: unknown, value: string, callback: (error?: Error) => void): void {
  if (!value) {
    callback(new Error('请输入IP地址'))
    return
  }
  const ipRegex = /^(\d{1,3}\.){3}\d{1,3}$/
  if (!ipRegex.test(value)) {
    callback(new Error('IP地址格式不正确'))
    return
  }
  const parts = value.split('.')
  for (const part of parts) {
    const num = Number(part)
    if (num < 0 || num > 255) {
      callback(new Error('IP地址每段应在0-255之间'))
      return
    }
  }
  callback()
}

/** 校验端口号 */
export function validatePort(_rule: unknown, value: string | number, callback: (error?: Error) => void): void {
  if (value === '' || value === null || value === undefined) {
    callback(new Error('请输入端口号'))
    return
  }
  const port = Number(value)
  if (!Number.isInteger(port) || port < 1 || port > 65535) {
    callback(new Error('端口号应在1-65535之间'))
    return
  }
  callback()
}

/** 校验非空字段 */
export function validateRequired(_rule: unknown, value: string, callback: (error?: Error) => void): void {
  if (!value || (typeof value === 'string' && value.trim() === '')) {
    callback(new Error('此字段不能为空'))
    return
  }
  callback()
}

/** 校验名称（字母、数字、中文、下划线、连字符，2-50位） */
export function validateName(_rule: unknown, value: string, callback: (error?: Error) => void): void {
  if (!value) {
    callback(new Error('请输入名称'))
    return
  }
  const nameRegex = /^[\u4e00-\u9fa5a-zA-Z0-9_-]{2,50}$/
  if (!nameRegex.test(value)) {
    callback(new Error('名称应为2-50位字母、数字、中文、下划线或连字符'))
    return
  }
  callback()
}

/** 校验正整数 */
export function validatePositiveInteger(_rule: unknown, value: string | number, callback: (error?: Error) => void): void {
  if (value === '' || value === null || value === undefined) {
    callback(new Error('请输入数值'))
    return
  }
  const num = Number(value)
  if (!Number.isInteger(num) || num <= 0) {
    callback(new Error('请输入正整数'))
    return
  }
  callback()
}
