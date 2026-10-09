import { CheckCircle, Loader2, QrCode } from "lucide-react"
import { useEffect, useState } from "react"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"

interface QRCodePaymentProps {
  qrCodeUrl: string
  provider: "alipay" | "wechat"
  transactionId: string
  onComplete?: () => void
  onCancel?: () => void
}

export function QRCodePayment({ qrCodeUrl, provider, transactionId, onComplete, onCancel }: QRCodePaymentProps) {
  const [status, setStatus] = useState<"pending" | "completed" | "failed">("pending")
  const [polling, setPolling] = useState(true)

  const providerName = provider === "alipay" ? "Alipay (支付宝)" : "WeChat Pay (微信支付)"

  // Task 12.7: Poll payment status
  useEffect(() => {
    if (!polling || status !== "pending") return

    const pollInterval = setInterval(async () => {
      try {
        const response = await fetch(`/api/v1/payment/${provider}/status/${transactionId}`)
        const data = await response.json()

        if (data.status === "completed") {
          setStatus("completed")
          setPolling(false)
          onComplete?.()
        } else if (data.status === "failed" || data.status === "cancelled") {
          setStatus("failed")
          setPolling(false)
        }
      } catch (error) {
        console.error("Failed to poll payment status:", error)
      }
    }, 3000) // Poll every 3 seconds

    return () => clearInterval(pollInterval)
  }, [polling, status, provider, transactionId, onComplete])

  if (status === "completed") {
    return (
      <Card>
        <CardContent className="p-8 text-center">
          <CheckCircle className="h-16 w-16 text-green-600 mx-auto mb-4" />
          <h3 className="text-xl font-semibold mb-2">Payment Successful!</h3>
          <p className="text-muted-foreground">Your license will be generated shortly.</p>
        </CardContent>
      </Card>
    )
  }

  if (status === "failed") {
    return (
      <Card>
        <CardContent className="p-8 text-center">
          <div className="rounded-full bg-red-100 p-3 w-16 h-16 mx-auto mb-4 flex items-center justify-center">
            <span className="text-2xl">❌</span>
          </div>
          <h3 className="text-xl font-semibold mb-2">Payment Failed</h3>
          <p className="text-muted-foreground mb-4">The payment was not completed.</p>
          {onCancel && (
            <Button variant="outline" onClick={onCancel}>
              Try Again
            </Button>
          )}
        </CardContent>
      </Card>
    )
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <QrCode className="h-5 w-5" />
          Scan to Pay with {providerName}
        </CardTitle>
        <CardDescription>Use your {providerName} app to scan this QR code</CardDescription>
      </CardHeader>
      <CardContent className="space-y-6">
        {/* Task 12.6: Display QR code */}
        <div className="flex justify-center">
          <div className="bg-white p-4 rounded-lg shadow-md">
            <img
              src={`https://api.qrserver.com/v1/create-qr-code/?size=256x256&data=${encodeURIComponent(qrCodeUrl)}`}
              alt="Payment QR Code"
              className="w-64 h-64"
            />
          </div>
        </div>

        <div className="text-center space-y-2">
          <div className="flex items-center justify-center gap-2 text-muted-foreground">
            <Loader2 className="h-4 w-4 animate-spin" />
            <span>Waiting for payment...</span>
          </div>
          <p className="text-xs text-muted-foreground">
            The page will automatically update once payment is confirmed
          </p>
        </div>

        <div className="rounded-md bg-blue-50 p-4 text-sm text-blue-900 dark:bg-blue-900/20 dark:text-blue-200">
          <p className="font-medium mb-1">Instructions:</p>
          <ol className="list-decimal list-inside space-y-1 text-xs">
            <li>Open your {providerName} app</li>
            <li>Scan the QR code above</li>
            <li>Confirm the payment amount</li>
            <li>Complete the payment</li>
          </ol>
        </div>

        {onCancel && (
          <Button variant="outline" className="w-full" onClick={onCancel}>
            Cancel Payment
          </Button>
        )}
      </CardContent>
    </Card>
  )
}
