import { useState, useRef, type ChangeEvent } from 'react'
import Papa from 'papaparse'
import * as XLSX from 'xlsx'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Upload, X, CheckCircle, AlertCircle } from 'lucide-react'

interface CsvUploadProps {
  columns: string[]
  onImport: (data: Record<string, string>[]) => void | Promise<void>
  accept?: string
  maxPreviewRows?: number
}

interface ValidationError {
  row: number
  message: string
}

export function CsvUpload({
  columns,
  onImport,
  accept = '.csv,.xlsx',
  maxPreviewRows = 10,
}: CsvUploadProps) {
  const [data, setData] = useState<Record<string, string>[]>([])
  const [errors, setErrors] = useState<ValidationError[]>([])
  const [fileName, setFileName] = useState('')
  const [importing, setImporting] = useState(false)
  const fileRef = useRef<HTMLInputElement>(null)

  const validate = (rows: Record<string, string>[]): ValidationError[] => {
    const errs: ValidationError[] = []
    if (rows.length === 0) {
      errs.push({ row: 0, message: '文件为空' })
      return errs
    }
    const headers = Object.keys(rows[0])
    for (const col of columns) {
      if (!headers.includes(col)) {
        errs.push({ row: 0, message: `缺少必需列: ${col}` })
      }
    }
    return errs
  }

  const handleFile = (e: ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return
    setFileName(file.name)

    if (file.name.endsWith('.xlsx') || file.name.endsWith('.xls')) {
      const reader = new FileReader()
      reader.onload = (evt) => {
        const wb = XLSX.read(evt.target?.result, { type: 'array' })
        const ws = wb.Sheets[wb.SheetNames[0]]
        const rows = XLSX.utils.sheet_to_json<Record<string, string>>(ws, { defval: '' })
        const errs = validate(rows)
        setData(rows)
        setErrors(errs)
      }
      reader.readAsArrayBuffer(file)
    } else {
      Papa.parse<Record<string, string>>(file, {
        header: true,
        skipEmptyLines: true,
        complete: (result) => {
          const rows = result.data
          const errs = validate(rows)
          setData(rows)
          setErrors(errs)
        },
      })
    }
  }

  const handleImport = async () => {
    setImporting(true)
    try {
      await onImport(data)
      reset()
    } finally {
      setImporting(false)
    }
  }

  const reset = () => {
    setData([])
    setErrors([])
    setFileName('')
    if (fileRef.current) fileRef.current.value = ''
  }

  const previewData = data.slice(0, maxPreviewRows)
  const previewHeaders = data.length > 0 ? Object.keys(data[0]) : []

  return (
    <div className="space-y-4">
      {/* Upload area */}
      <div className="flex items-center gap-3">
        <Input
          ref={fileRef}
          type="file"
          accept={accept}
          onChange={handleFile}
          className="max-w-sm"
        />
        {fileName && (
          <Button variant="ghost" size="icon" onClick={reset}>
            <X className="h-4 w-4" />
          </Button>
        )}
      </div>

      {/* Validation errors */}
      {errors.length > 0 && (
        <div className="rounded-md border border-destructive/50 bg-destructive/5 p-3 space-y-1">
          {errors.map((err, i) => (
            <div key={i} className="flex items-center gap-2 text-sm text-destructive">
              <AlertCircle className="h-4 w-4 shrink-0" />
              <span>{err.message}</span>
            </div>
          ))}
        </div>
      )}

      {/* Preview table */}
      {data.length > 0 && errors.length === 0 && (
        <>
          <div className="flex items-center gap-2 text-sm text-muted-foreground">
            <CheckCircle className="h-4 w-4 text-green-600" />
            <span>解析成功，共 {data.length} 条记录（预览前 {Math.min(data.length, maxPreviewRows)} 条）</span>
          </div>
          <div className="rounded-md border max-h-80 overflow-auto">
            <Table>
              <TableHeader>
                <TableRow>
                  {previewHeaders.map((h) => (
                    <TableHead key={h}>{h}</TableHead>
                  ))}
                </TableRow>
              </TableHeader>
              <TableBody>
                {previewData.map((row, i) => (
                  <TableRow key={i}>
                    {previewHeaders.map((h) => (
                      <TableCell key={h} className="text-sm">
                        {row[h]}
                      </TableCell>
                    ))}
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
          <Button onClick={handleImport} disabled={importing}>
            <Upload className="mr-2 h-4 w-4" />
            {importing ? '导入中...' : `确认导入 ${data.length} 条记录`}
          </Button>
        </>
      )}
    </div>
  )
}
