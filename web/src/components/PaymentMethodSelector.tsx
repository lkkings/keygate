import { useQuery } from "@tanstack/react-query"
import { CreditCard, Loader2 } from "lucide-react"
import { useState } from "react"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { Label } from "@/components/ui/label"
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group"
import type { Plan } from "@/lib/api"

interface PaymentMethodSelectorProps {
  plan: Plan
  onCheckout: (provider: string) => void
  loading?: boolean
}

export function PaymentMethodSelector({ plan, onCheckout, loading }: PaymentMethodSelectorProps) {
  const [selectedProvider, setSelectedProvider] = useState<string>("")

  // Fetch available providers (Task 12.1)
  const { data: providersData, isLoading } = useQuery({
    queryKey: ["payment-providers"],
    queryFn: () => fetch("/api/v1/payment/providers").then((r) => r.json()),
  })

  // Task 12.2: Filter providers based on plan's available currencies
  const getAvailableProviders = () => {
    if (!providersData?.providers) return []

    const providers = providersData.providers as string[]
    const hasCNY = plan.price_cny != null && plan.price_cny > 0
    const hasUSD = plan.price_usd != null && plan.price_usd > 0
    const hasHKD = plan.price_hkd != null && plan.price_hkd > 0

    return providers.filter((provider) => {
      switch (provider) {
        case "stripe":
          // Stripe supports USD
          return hasUSD
        case "alipay":
          // Alipay requires CNY pricing
          return hasCNY
        case "wechat":
          // WeChat Pay requires CNY pricing
          return hasCNY
        case "epay":
          // ePay can support multiple currencies (for now, require any price)
          return hasUSD || hasCNY || hasHKD
        default:
          return false
      }
    })
  }

  const availableProviders = getAvailableProviders()

  // Task 12.3: Provider metadata with logos and labels
  const providerInfo: Record<
    string,
    { label: string; labelCn: string; description: string; icon: string }
  > = {
    stripe: {
      label: "Stripe",
      labelCn: "Stripe",
      description: "Credit card, debit card, international payments",
      icon: "💳",
    },
    alipay: {
      label: "Alipay",
      labelCn: "支付宝",
      description: "Alipay wallet payments (CNY)",
      icon: "🇨🇳",
    },
    wechat: {
      label: "WeChat Pay",
      labelCn: "微信支付",
      description: "WeChat QR code payment (CNY)",
      icon: "💬",
    },
    epay: {
      label: "ePay",
      labelCn: "ePay",
      description: "Alternative payment gateway",
      icon: "💰",
    },
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center p-8">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    )
  }

  if (availableProviders.length === 0) {
    return (
      <Card>
        <CardContent className="p-6">
          <p className="text-center text-muted-foreground">
            No payment methods available for this plan. Please contact support.
          </p>
        </CardContent>
      </Card>
    )
  }

  return (
    <div className="space-y-6">
      <div>
        <h3 className="text-lg font-semibold mb-2">Select Payment Method</h3>
        <p className="text-sm text-muted-foreground">Choose how you'd like to pay for your license</p>
      </div>

      {/* Task 12.4: Provider selection with radio buttons */}
      <RadioGroup value={selectedProvider} onValueChange={setSelectedProvider}>
        <div className="space-y-3">
          {availableProviders.map((provider) => {
            const info = providerInfo[provider]
            return (
              <Card
                key={provider}
                className={`cursor-pointer transition-colors ${
                  selectedProvider === provider ? "border-primary bg-primary/5" : "hover:border-primary/50"
                }`}
                onClick={() => setSelectedProvider(provider)}
              >
                <CardContent className="p-4">
                  <div className="flex items-start gap-4">
                    <RadioGroupItem value={provider} id={provider} className="mt-1" />
                    <div className="flex-1">
                      <Label htmlFor={provider} className="cursor-pointer">
                        <div className="flex items-center gap-2 mb-1">
                          <span className="text-2xl">{info.icon}</span>
                          <span className="font-semibold">{info.label}</span>
                          {info.labelCn !== info.label && (
                            <span className="text-muted-foreground">({info.labelCn})</span>
                          )}
                        </div>
                        <p className="text-sm text-muted-foreground">{info.description}</p>
                      </Label>
                    </div>
                  </div>
                </CardContent>
              </Card>
            )
          })}
        </div>
      </RadioGroup>

      {/* Task 12.4: Continue button disabled until provider selected */}
      <Button
        size="lg"
        className="w-full"
        disabled={!selectedProvider || loading}
        onClick={() => onCheckout(selectedProvider)}
      >
        {loading ? (
          <>
            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            Processing...
          </>
        ) : (
          <>
            <CreditCard className="mr-2 h-4 w-4" />
            Continue to Payment
          </>
        )}
      </Button>
    </div>
  )
}
