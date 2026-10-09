import { useState } from "react"
import { useMutation, useQueryClient } from "@tanstack/react-query"
import { AlertTriangle, DollarSign, Loader2 } from "lucide-react"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Checkbox } from "@/components/ui/checkbox"
import { showToast } from "@/components/toast"

interface RefundModalProps {
  open: boolean
  onClose: () => void
  transactionId: string
  provider: string
  amount: number
  currency: string
  customerEmail: string
}

// Task 16.2: Refund confirmation modal with amount input
export function RefundModal({
  open,
  onClose,
  transactionId,
  provider,
  amount,
  currency,
  customerEmail,
}: RefundModalProps) {
  const queryClient = useQueryClient()
  const [refundAmount, setRefundAmount] = useState(amount.toString())
  const [reason, setReason] = useState("")
  const [revokeLicense, setRevokeLicense] = useState(false)
  const isFullRefund = parseInt(refundAmount) === amount

  // Task 16.3: Call refund API
  const refundMutation = useMutation({
    mutationFn: async () => {
      const response = await fetch(`/api/v1/payment/${provider}/refund`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          transaction_id: transactionId,
          amount: parseInt(refundAmount),
          reason: reason,
          revoke_license: revokeLicense,
        }),
      })

      if (!response.ok) {
        const error = await response.json()
        throw new Error(error.error || "Refund failed")
      }

      return response.json()
    },
    onSuccess: () => {
      showToast("Refund processed successfully", "success")
      queryClient.invalidateQueries({ queryKey: ["transactions"] })
      queryClient.invalidateQueries({ queryKey: ["licenses"] })
      onClose()
    },
    onError: (error: Error) => {
      showToast(`Refund failed: ${error.message}`, "error")
    },
  })

  const handleRefund = () => {
    if (!refundAmount || parseInt(refundAmount) <= 0) {
      showToast("Please enter a valid refund amount", "error")
      return
    }

    if (parseInt(refundAmount) > amount) {
      showToast("Refund amount cannot exceed transaction amount", "error")
      return
    }

    refundMutation.mutate()
  }

  const formatAmount = (cents: number) => {
    return (cents / 100).toFixed(2)
  }

  return (
    <Dialog open={open} onOpenChange={onClose}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Process Refund</DialogTitle>
          <DialogDescription>
            Refund payment to customer. This action cannot be undone.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          {/* Transaction details */}
          <div className="rounded-lg bg-muted p-4 space-y-2 text-sm">
            <div className="flex justify-between">
              <span className="text-muted-foreground">Transaction ID:</span>
              <span className="font-mono text-xs">{transactionId}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-muted-foreground">Customer:</span>
              <span>{customerEmail}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-muted-foreground">Original Amount:</span>
              <span className="font-semibold">
                {currency} {formatAmount(amount)}
              </span>
            </div>
            <div className="flex justify-between">
              <span className="text-muted-foreground">Provider:</span>
              <span className="capitalize">{provider}</span>
            </div>
          </div>

          {/* Refund amount input */}
          <div className="space-y-2">
            <Label htmlFor="refund-amount">Refund Amount ({currency})</Label>
            <div className="relative">
              <DollarSign className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
              <Input
                id="refund-amount"
                type="number"
                step="0.01"
                value={refundAmount}
                onChange={(e) => setRefundAmount(e.target.value)}
                className="pl-9"
                placeholder={formatAmount(amount)}
              />
            </div>
            <div className="flex gap-2">
              <Button
                type="button"
                size="sm"
                variant="outline"
                onClick={() => setRefundAmount(amount.toString())}
                disabled={isFullRefund}
              >
                Full Refund
              </Button>
              <Button
                type="button"
                size="sm"
                variant="outline"
                onClick={() => setRefundAmount(Math.floor(amount / 2).toString())}
              >
                50%
              </Button>
            </div>
          </div>

          {/* Reason */}
          <div className="space-y-2">
            <Label htmlFor="reason">Reason (Optional)</Label>
            <Input
              id="reason"
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              placeholder="Customer requested refund"
            />
          </div>

          {/* Task 16.5: Revoke license option */}
          <div className="flex items-start space-x-2">
            <Checkbox
              id="revoke"
              checked={revokeLicense}
              onCheckedChange={(checked: boolean) => setRevokeLicense(checked)}
            />
            <div className="space-y-1 leading-none">
              <Label htmlFor="revoke" className="cursor-pointer">
                Revoke license immediately
              </Label>
              <p className="text-xs text-muted-foreground">
                The license will be disabled and cannot be used
              </p>
            </div>
          </div>

          {/* Warning */}
          <div className="flex gap-2 p-3 bg-yellow-50 border border-yellow-200 rounded-md dark:bg-yellow-900/20 dark:border-yellow-900">
            <AlertTriangle className="h-5 w-5 text-yellow-600 dark:text-yellow-500 flex-shrink-0" />
            <div className="text-sm text-yellow-800 dark:text-yellow-200">
              <p className="font-medium">This action cannot be undone</p>
              <p className="text-xs mt-1">
                The refund will be processed through {provider} and funds will be returned to the customer.
              </p>
            </div>
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={onClose} disabled={refundMutation.isPending}>
            Cancel
          </Button>
          <Button
            onClick={handleRefund}
            disabled={refundMutation.isPending}
            variant="destructive"
          >
            {refundMutation.isPending ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                Processing...
              </>
            ) : (
              "Process Refund"
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
