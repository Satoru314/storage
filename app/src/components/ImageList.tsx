import { useState, useEffect } from 'react'

interface ImageMeta {
  id: string
  objectKey: string
  originalName: string
  mimeType: string
  byteSize: number
  uploadedAt?: string
}

interface ViewUrlRequest {
  id: string
}

interface ViewUrlResult {
  id: string
  url: string
  expiresAt: string
}

interface ViewUrlsResponse {
  results: ViewUrlResult[]
}

interface ImageListResponse {
  items: ImageMeta[]
  nextCursor?: string
}

const ImageList = () => {
  const [images, setImages] = useState<ImageMeta[]>([])
  const [imageUrls, setImageUrls] = useState<Map<string, string>>(new Map())
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    fetchImages()
  }, [])

  const fetchImages = async () => {
    try {
      setLoading(true)
      const response = await fetch('http://localhost:8080/images')

      if (!response.ok) {
        throw new Error('Failed to fetch images')
      }

      const data: ImageListResponse = await response.json()
      setImages(data.items)

      // Fetch presigned URLs for images
      if (data.items.length > 0) {
        await fetchImageUrls(data.items)
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error')
    } finally {
      setLoading(false)
    }
  }

  const fetchImageUrls = async (imageList: ImageMeta[]) => {
    try {
      const requests: ViewUrlRequest[] = imageList.map(image => ({ id: image.id }))

      const response = await fetch('http://localhost:8080/images/view-urls', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ requests })
      })

      if (!response.ok) {
        throw new Error('Failed to fetch image URLs')
      }

      const data: ViewUrlsResponse = await response.json()
      const urlMap = new Map<string, string>()

      data.results.forEach(result => {
        urlMap.set(result.id, result.url)
      })

      setImageUrls(urlMap)
    } catch (err) {
      console.error('Error fetching image URLs:', err)
    }
  }

  const formatFileSize = (bytes: number) => {
    const sizes = ['B', 'KB', 'MB', 'GB']
    if (bytes === 0) return '0 B'
    const i = Math.floor(Math.log(bytes) / Math.log(1024))
    return Math.round(bytes / Math.pow(1024, i) * 100) / 100 + ' ' + sizes[i]
  }

  const formatDate = (dateString?: string) => {
    if (!dateString) return 'N/A'
    return new Date(dateString).toLocaleDateString()
  }

  if (loading) {
    return (
      <div className="flex justify-center items-center h-64">
        <div className="text-gray-500">Loading images...</div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="bg-red-50 border border-red-200 rounded-md p-4">
        <div className="text-red-800">Error: {error}</div>
        <button
          onClick={fetchImages}
          className="mt-2 bg-red-600 hover:bg-red-700 text-white px-4 py-2 rounded-md text-sm"
        >
          Retry
        </button>
      </div>
    )
  }

  if (images.length === 0) {
    return (
      <div className="text-center py-12">
        <div className="text-gray-500 text-lg mb-4">No images uploaded yet</div>
        <p className="text-gray-400 mb-6">Upload your first image to get started</p>
      </div>
    )
  }

  return (
    <div>
      <div className="mb-6">
        <h2 className="text-2xl font-bold text-gray-900">Images ({images.length})</h2>
      </div>

      <div className="bg-white shadow overflow-hidden sm:rounded-md">
        <ul className="divide-y divide-gray-200">
          {images.map((image) => (
            <li key={image.id} className="px-6 py-4">
              <div className="flex items-center space-x-4">
                <div className="flex-shrink-0">
                  {imageUrls.has(image.id) && image.mimeType.startsWith('image/') ? (
                    <img
                      src={imageUrls.get(image.id)}
                      alt={image.originalName}
                      className="w-16 h-16 object-cover rounded-lg border border-gray-200"
                    />
                  ) : (
                    <div className="w-16 h-16 bg-gray-100 rounded-lg border border-gray-200 flex items-center justify-center">
                      <span className="text-gray-400 text-xs">No Preview</span>
                    </div>
                  )}
                </div>
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-medium text-gray-900 truncate">
                    {image.originalName}
                  </p>
                  <p className="text-sm text-gray-500">
                    {image.mimeType} • {formatFileSize(image.byteSize)}
                  </p>
                </div>
                <div className="flex-shrink-0 text-right">
                  <p className="text-sm text-gray-500">
                    Uploaded {formatDate(image.uploadedAt)}
                  </p>
                  {imageUrls.has(image.id) && (
                    <a
                      href={imageUrls.get(image.id)}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="inline-block mt-1 text-xs text-blue-600 hover:text-blue-800"
                    >
                      View Full Size
                    </a>
                  )}
                </div>
              </div>
            </li>
          ))}
        </ul>
      </div>
    </div>
  )
}

export default ImageList