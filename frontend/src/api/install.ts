import request from '@/utils/request'

/** 安装命令响应 */
export interface InstallCommandData {
  command: string
  server: string
  token: string
}

/** 获取Agent安装命令 */
export function getInstallCommandApi(): Promise<InstallCommandData> {
  return request.get('/install/command').then((res) => res.data.data)
}
