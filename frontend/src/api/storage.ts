import request from '@/utils/request'
import axios from 'axios'

export interface StorageFile {
  id: number
  name: string
  path: string
  size: number
  md5: string
  mime_type: string
  created_at: string
  updated_at: string
}

export interface ResumeInfo {
  file_name: string
  file_exists: boolean
  uploaded_bytes: number
  uploaded_chunks: number
  total_chunks: number
}

/** 获取所有存储文件 */
export function getStorageFilesApi(keyword?: string): Promise<StorageFile[]> {
  return request.get('/storage/files', { params: { keyword } }).then((res) => res.data.data)
}

/** 获取文件详情 */
export function getStorageFileApi(id: number): Promise<StorageFile> {
  return request.get(`/storage/files/${id}`).then((res) => res.data.data)
}

/** 删除文件 */
export function deleteStorageFileApi(id: number): Promise<void> {
  return request.delete(`/storage/files/${id}`).then((res) => res.data)
}

/** 获取续传信息 */
export function getResumeInfoApi(fileName: string): Promise<ResumeInfo> {
  return request.get('/storage/resume-info', { params: { file_name: fileName } }).then((res) => res.data.data)
}

/** 上传文件分块（带进度回调） */
export function uploadChunkApi(
  fileName: string,
  chunkIndex: number,
  totalChunks: number,
  chunk: Blob,
  onProgress?: (percent: number) => void
): Promise<{ status: string; next_chunk?: number }> {
  const formData = new FormData()
  formData.append('file', chunk)
  formData.append('file_name', fileName)
  formData.append('chunk_index', String(chunkIndex))
  formData.append('total_chunks', String(totalChunks))

  return request
    .post('/storage/upload', formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
      onUploadProgress: (e) => {
        if (onProgress && e.total) {
          onProgress(Math.round((e.loaded / e.total) * 100))
        }
      },
    })
    .then((res) => res.data.data)
}

/** 获取下载链接（不在 /api/v1 下） */
export function getDownloadUrl(path: string): string {
  // 下载路由是 /download/*path，不在 /api/v1 下
  const base = window.location.origin
  return `${base}/download/${encodeURIComponent(path)}`
}

/** 获取 wget 命令（带 --no-check-certificate 应对自签证书） */
export function getWgetCommand(path: string): string {
  return `wget --no-check-certificate "${getDownloadUrl(path)}"`
}
