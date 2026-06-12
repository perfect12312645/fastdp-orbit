import { format } from 'date-fns'

/**
 * 格式化日期时间
 * @param date 日期字符串或Date对象
 * @param pattern 格式化模式，默认 YYYY-MM-DD HH:mm:ss
 */
export function formatDateTime(date: string | Date | null | undefined, pattern = 'YYYY-MM-DD HH:mm:ss'): string {
  if (!date) return '-'
  try {
    const d = typeof date === 'string' ? new Date(date) : date
    return format(d, pattern)
  } catch {
    return '-'
  }
}

/**
 * 格式化文件大小
 * @param bytes 字节数
 */
export function formatFileSize(bytes: number | null | undefined): string {
  if (bytes === null || bytes === undefined) return '-'
  if (bytes === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB', 'PB']
  const k = 1024
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return `${(bytes / Math.pow(k, i)).toFixed(2)} ${units[i]}`
}

/**
 * 格式化内存大小（MB/GB）
 */
export function formatMemory(mb: number | null | undefined): string {
  if (mb === null || mb === undefined) return '-'
  if (mb >= 1024) {
    return `${(mb / 1024).toFixed(1)} GB`
  }
  return `${mb} MB`
}

/**
 * 深拷贝对象
 */
export function deepClone<T>(obj: T): T {
  return JSON.parse(JSON.stringify(obj))
}
