import { useState, useRef } from 'react'

interface UploadResponse {
  image: {
    id: string
    objectKey: string
    originalName: string
    mimeType: string
    byteSize: number
    status: string
  }
  upload: {
    method: string
    url: string
    headers: Record<string, string>
    expiresInSec: number
  }
}

const Upload = () => {
  const [selectedFile, setSelectedFile] = useState<File | null>(null)
  const [uploading, setUploading] = useState(false)
  const [progress, setProgress] = useState(0)
  const [message, setMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)

  const allowedTypes = ['image/jpeg', 'image/png', 'image/webp', 'image/heic']
  const maxSize = 50 * 1024 * 1024 // 50MB

  const handleFileSelect = (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0]
    if (!file) return

    if (!allowedTypes.includes(file.type)) {
      setMessage({ type: 'error', text: 'Please select a valid image file (JPEG, PNG, WebP, HEIC)' })
      return
    }

    if (file.size > maxSize) {
      setMessage({ type: 'error', text: 'File size must be less than 50MB' })
      return
    }

    setSelectedFile(file)
    setMessage(null)
  }

  const handleUpload = async () => {
    if (!selectedFile) return

    try {
      setUploading(true)
      setProgress(0)
      setMessage(null)

      // Step 1: Request upload URL
      const uploadRequest = {
        fileName: selectedFile.name,
        contentType: selectedFile.type,
        fileSize: selectedFile.size,
      }

      const response = await fetch('http://localhost:8080/images/upload-request', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(uploadRequest),
      })

      if (!response.ok) {
        throw new Error('Failed to get upload URL')
      }

      const data: UploadResponse = await response.json()
      
      setProgress(25)

      // Step 2: Upload to S3
      const uploadResponse = await fetch(data.upload.url, {
        method: 'PUT',
        headers: data.upload.headers,
        body: selectedFile,
      })

      if (!uploadResponse.ok) {
        throw new Error('Failed to upload file to S3')
      }

      setProgress(75)

      // Step 3: Complete upload
      const completeRequest = {
        id: data.image.id,
        objectKey: data.image.objectKey,
      }

      const completeResponse = await fetch('http://localhost:8080/images/upload-complete', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(completeRequest),
      })

      if (!completeResponse.ok) {
        throw new Error('Failed to complete upload')
      }

      setProgress(100)
      setMessage({ type: 'success', text: 'Upload completed successfully!' })
      setSelectedFile(null)
      if (fileInputRef.current) {
        fileInputRef.current.value = ''
      }

    } catch (error) {
      setMessage({
        type: 'error',
        text: error instanceof Error ? error.message : 'Upload failed',
      })
    } finally {
      setUploading(false)
      setProgress(0)
    }
  }

  const formatFileSize = (bytes: number) => {
    const sizes = ['B', 'KB', 'MB', 'GB']
    if (bytes === 0) return '0 B'
    const i = Math.floor(Math.log(bytes) / Math.log(1024))
    return Math.round(bytes / Math.pow(1024, i) * 100) / 100 + ' ' + sizes[i]
  }

  return (
    <div className="max-w-2xl mx-auto">
      <div className="mb-6">
        <h2 className="text-2xl font-bold text-gray-900">Upload Image</h2>
        <p className="text-gray-600 mt-2">
          Select an image file to upload. Supported formats: JPEG, PNG, WebP, HEIC (max 50MB)
        </p>
      </div>

      <div className="bg-white shadow-sm rounded-lg p-6">
        <div className="space-y-6">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Select Image File
            </label>
            <input
              ref={fileInputRef}
              type="file"
              accept="image/*"
              onChange={handleFileSelect}
              disabled={uploading}
              className="block w-full text-sm text-gray-500
                         file:mr-4 file:py-2 file:px-4
                         file:rounded-full file:border-0
                         file:text-sm file:font-semibold
                         file:bg-blue-50 file:text-blue-700
                         hover:file:bg-blue-100"
            />
          </div>

          {selectedFile && (
            <div className="bg-gray-50 rounded-md p-4">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm font-medium text-gray-900">{selectedFile.name}</p>
                  <p className="text-sm text-gray-500">
                    {selectedFile.type} • {formatFileSize(selectedFile.size)}
                  </p>
                </div>
                <button
                  onClick={() => setSelectedFile(null)}
                  disabled={uploading}
                  className="text-red-600 hover:text-red-800 text-sm"
                >
                  Remove
                </button>
              </div>
            </div>
          )}

          {uploading && (
            <div>
              <div className="flex justify-between mb-2">
                <span className="text-sm text-gray-700">Uploading...</span>
                <span className="text-sm text-gray-700">{progress}%</span>
              </div>
              <div className="w-full bg-gray-200 rounded-full h-2">
                <div
                  className="bg-blue-600 h-2 rounded-full transition-all duration-300"
                  style={{ width: `${progress}%` }}
                />
              </div>
            </div>
          )}

          {message && (
            <div
              className={`rounded-md p-4 ${
                message.type === 'success'
                  ? 'bg-green-50 border border-green-200 text-green-800'
                  : 'bg-red-50 border border-red-200 text-red-800'
              }`}
            >
              {message.text}
            </div>
          )}

          <div className="flex gap-4">
            <button
              onClick={handleUpload}
              disabled={!selectedFile || uploading}
              className="flex-1 bg-blue-600 hover:bg-blue-700 disabled:bg-gray-300 
                         text-white font-medium py-2 px-4 rounded-md
                         transition-colors duration-200"
            >
              {uploading ? 'Uploading...' : 'Upload Image'}
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}

export default Upload