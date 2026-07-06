import * as XLSX from 'xlsx'
import { ElMessage } from 'element-plus'

/** 导出配置 */
export interface ExportOption {
  /** 导出文件名（不含后缀） */
  filename: string
  /** 表头字段映射 { 列名: 数据字段名 } */
  columns: Record<string, string>
  /** 数据 */
  data: Record<string, unknown>[]
}

/**
 * 导出数据为Excel文件
 * @param option 导出配置
 */
export function exportToExcel(option: ExportOption): void {
  try {
    const { filename, columns, data } = option
    const headerKeys = Object.keys(columns)
    const headerValues = Object.values(columns)

    // 构建表头
    const sheetData: unknown[][] = [headerValues]

    // 填充数据行
    data.forEach((row) => {
      const rowArr = headerKeys.map((key) => row[key] ?? '')
      sheetData.push(rowArr)
    })

    const worksheet = XLSX.utils.aoa_to_sheet(sheetData)

    // 设置列宽
    worksheet['!cols'] = headerKeys.map(() => ({ wch: 20 }))

    const workbook = XLSX.utils.book_new()
    XLSX.utils.book_append_sheet(workbook, worksheet, 'Sheet1')
    XLSX.writeFile(workbook, `${filename}.xlsx`)

    ElMessage.success('导出成功')
  } catch {
    ElMessage.error('导出失败，请稍后重试')
  }
}
