import { useState } from "react"
import { useQuery } from "@tanstack/react-query"
import { Download, Filter } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  DataTable,
  DataTableBody,
  DataTableCell,
  DataTableHead,
  DataTableHeader,
  DataTablePagination,
  DataTableRow,
} from "@/components/ui/data-table"
import { Badge } from "@/components/ui/badge"

interface Transaction {
  id: number
  provider_name: string
  provider_tx_id: string
  amount: number
  currency: string
  status: string
  customer_email: string
  refund_amount?: number
  created_at: string
  refunded_at?: string
}

// Task 17.3: Transaction list UI
export default function TransactionsPage() {
  const [page, setPage] = useState(1)
  const [provider, setProvider] = useState("")
  const [status, setStatus] = useState("")
  const [dateFrom, setDateFrom] = useState("")
  const [dateTo, setDateTo] = useState("")

  // Task 17.1: Fetch paginated transactions with filters
  const { data, isLoading } = useQuery({
    queryKey: ["transactions", page, provider, status, dateFrom, dateTo],
    queryFn: async () => {
      const params = new URLSearchParams({
        page: page.toString(),
        per_page: "50",
      })
      if (provider) params.append("provider", provider)
      if (status) params.append("status", status)
      if (dateFrom) params.append("date_from", dateFrom)
      if (dateTo) params.append("date_to", dateTo)

      const response = await fetch(`/api/v1/admin/transactions?${params}`)
      return response.json()
    },
  })

  // Task 17.4: Revenue report
  const { data: revenueData } = useQuery({
    queryKey: ["revenue", dateFrom, dateTo],
    queryFn: async () => {
      const params = new URLSearchParams()
      if (dateFrom) params.append("date_from", dateFrom)
      if (dateTo) params.append("date_to", dateTo)

      const response = await fetch(`/api/v1/admin/transactions/revenue?${params}`)
      return response.json()
    },
  })

  // Task 17.5: CSV export
  const handleExport = () => {
    const params = new URLSearchParams()
    if (provider) params.append("provider", provider)
    if (status) params.append("status", status)
    if (dateFrom) params.append("date_from", dateFrom)
    if (dateTo) params.append("date_to", dateTo)

    window.location.href = `/api/v1/admin/transactions/export?${params}`
  }

  const formatAmount = (cents: number, currency: string) => {
    return `${currency} ${(cents / 100).toFixed(2)}`
  }

  const getStatusBadge = (status: string) => {
    const variants: Record<string, "default" | "secondary" | "destructive" | "outline"> = {
      completed: "default",
      pending: "secondary",
      failed: "destructive",
      refunded: "outline",
    }
    return <Badge variant={variants[status] || "default"}>{status}</Badge>
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold">Transactions</h1>
        <p className="text-muted-foreground">View and manage payment transactions</p>
      </div>

      {/* Task 17.4: Revenue summary */}
      {revenueData?.revenue_by_currency && (
        <div className="grid gap-4 md:grid-cols-3">
          {Object.entries(revenueData.revenue_by_currency).map(([currency, data]: [string, any]) => (
            <Card key={currency}>
              <CardHeader className="pb-3">
                <CardDescription>{currency} Revenue</CardDescription>
                <CardTitle className="text-2xl">
                  {currency} {data.total_formatted}
                </CardTitle>
              </CardHeader>
              <CardContent>
                <p className="text-xs text-muted-foreground">
                  {data.transaction_count} transactions
                </p>
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      {/* Task 17.2: Filters */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Filter className="h-5 w-5" />
            Filters
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
            <div className="space-y-2">
              <Label>Provider</Label>
              <Select value={provider} onValueChange={setProvider}>
                <SelectTrigger>
                  <SelectValue placeholder="All providers" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="">All providers</SelectItem>
                  <SelectItem value="stripe">Stripe</SelectItem>
                  <SelectItem value="alipay">Alipay</SelectItem>
                  <SelectItem value="wechat">WeChat Pay</SelectItem>
                  <SelectItem value="epay">ePay</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-2">
              <Label>Status</Label>
              <Select value={status} onValueChange={setStatus}>
                <SelectTrigger>
                  <SelectValue placeholder="All statuses" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="">All statuses</SelectItem>
                  <SelectItem value="completed">Completed</SelectItem>
                  <SelectItem value="pending">Pending</SelectItem>
                  <SelectItem value="failed">Failed</SelectItem>
                  <SelectItem value="refunded">Refunded</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-2">
              <Label>Date From</Label>
              <Input
                type="date"
                value={dateFrom}
                onChange={(e) => setDateFrom(e.target.value)}
              />
            </div>

            <div className="space-y-2">
              <Label>Date To</Label>
              <Input
                type="date"
                value={dateTo}
                onChange={(e) => setDateTo(e.target.value)}
              />
            </div>
          </div>

          <div className="mt-4 flex gap-2">
            <Button
              variant="outline"
              onClick={() => {
                setProvider("")
                setStatus("")
                setDateFrom("")
                setDateTo("")
              }}
            >
              Clear Filters
            </Button>
            <Button variant="outline" onClick={handleExport}>
              <Download className="mr-2 h-4 w-4" />
              Export CSV
            </Button>
          </div>
        </CardContent>
      </Card>

      {/* Task 17.3: Transaction table */}
      <Card>
        <CardHeader>
          <CardTitle>Transaction History</CardTitle>
        </CardHeader>
        <CardContent>
          <DataTable>
            <DataTableHeader>
              <DataTableRow>
                <DataTableHead>Provider</DataTableHead>
                <DataTableHead>Transaction ID</DataTableHead>
                <DataTableHead>Amount</DataTableHead>
                <DataTableHead>Status</DataTableHead>
                <DataTableHead>Customer</DataTableHead>
                <DataTableHead>Date</DataTableHead>
              </DataTableRow>
            </DataTableHeader>
            <DataTableBody>
              {!isLoading && data?.transactions?.map((txn: Transaction) => (
                <DataTableRow key={txn.id}>
                  <DataTableCell className="capitalize">{txn.provider_name}</DataTableCell>
                  <DataTableCell className="font-mono text-xs">{txn.provider_tx_id}</DataTableCell>
                  <DataTableCell>
                    {formatAmount(txn.amount, txn.currency)}
                    {txn.refund_amount && txn.refund_amount > 0 && (
                      <span className="text-xs text-muted-foreground ml-2">
                        (Refunded: {formatAmount(txn.refund_amount, txn.currency)})
                      </span>
                    )}
                  </DataTableCell>
                  <DataTableCell>{getStatusBadge(txn.status)}</DataTableCell>
                  <DataTableCell>{txn.customer_email}</DataTableCell>
                  <DataTableCell>
                    {new Date(txn.created_at).toLocaleDateString()}
                  </DataTableCell>
                </DataTableRow>
              ))}
            </DataTableBody>
          </DataTable>

          {data?.pagination && (
            <DataTablePagination
              page={page}
              totalPages={Math.ceil(data.pagination.total / data.pagination.per_page)}
              total={data.pagination.total}
              pageSize={data.pagination.per_page}
              onPageChange={setPage}
            />
          )}
        </CardContent>
      </Card>
    </div>
  )
}
