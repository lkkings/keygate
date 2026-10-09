import { useEffect, useState } from "react"
import { useSearchParams } from "react-router-dom"
import { useMutation, useQuery } from "@tanstack/react-query"
import { Loader2, RefreshCw, ArrowUp } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Label } from "@/components/ui/label"
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group"
import { PaymentMethodSelector } from "@/components/PaymentMethodSelector"
import { showToast } from "@/components/toast"
import type { Plan } from "@/lib/api"

interface RenewalData {
  license: {
    id: string
    email: string
    plan_id: string
    valid_until: string
    status: string
  }
  current_plan: Plan
  customer: {
    email: string
  }
  available_plans?: Plan[] // Task 15.3: For upgrade option
}

export function RenewalPage() {
  const [searchParams] = useSearchParams()
  const licenseID = searchParams.get("license")
  const [selectedPlanID, setSelectedPlanID] = useState<string>("")
  const [showPaymentSelector, setShowPaymentSelector] = useState(false)

  // Task 15.1: Load renewal data
  const { data: renewalData, isLoading } = useQuery<RenewalData>({
    queryKey: ["renewal", licenseID],
    queryFn: () => fetch(`/api/v1/renewal?license=${licenseID}`).then((r) => r.json()),
    enabled: !!licenseID,
  })

  // Task 15.2: Pre-select current plan
  useEffect(() => {
    if (renewalData?.current_plan) {
      setSelectedPlanID(renewalData.current_plan.id)
    }
  }, [renewalData])

  const checkoutMutation = useMutation({
    mutationFn: (provider: string) =>
      fetch("/api/v1/payment/renewal/checkout", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          license_id: licenseID,
          plan_id: selectedPlanID,
          provider: provider,
        }),
      }).then((r) => r.json()),
    onSuccess: (data) => {
      if (data.checkout_url) {
        window.location.href = data.checkout_url
      }
    },
    onError: (error: Error) => {
      showToast(`Failed to create checkout: ${error.message}`, "error")
    },
  })

  if (!licenseID) {
    return (
      <div className="container max-w-2xl py-8">
        <Card>
          <CardContent className="p-8 text-center">
            <p className="text-muted-foreground">Invalid renewal link</p>
          </CardContent>
        </Card>
      </div>
    )
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    )
  }

  if (!renewalData) {
    return (
      <div className="container max-w-2xl py-8">
        <Card>
          <CardContent className="p-8 text-center">
            <p className="text-muted-foreground">License not found</p>
          </CardContent>
        </Card>
      </div>
    )
  }

  const isUpgrade = selectedPlanID !== renewalData.current_plan.id

  return (
    <div className="container max-w-3xl py-8 space-y-6">
      <div>
        <h1 className="text-3xl font-bold">Renew Your License</h1>
        <p className="text-muted-foreground mt-2">
          Extend your license to continue using the product
        </p>
      </div>

      {/* Task 15.2: Display current license info */}
      <Card>
        <CardHeader>
          <CardTitle>Current License</CardTitle>
        </CardHeader>
        <CardContent className="space-y-2">
          <div className="grid grid-cols-2 gap-2 text-sm">
            <span className="text-muted-foreground">Email:</span>
            <span className="font-medium">{renewalData.license.email}</span>

            <span className="text-muted-foreground">Plan:</span>
            <span className="font-medium">{renewalData.current_plan.name}</span>

            <span className="text-muted-foreground">Expires:</span>
            <span className="font-medium">
              {new Date(renewalData.license.valid_until).toLocaleDateString()}
            </span>
          </div>
        </CardContent>
      </Card>

      {/* Task 15.3: Plan selection with upgrade option */}
      <Card>
        <CardHeader>
          <CardTitle>Select Plan</CardTitle>
          <CardDescription>
            Renew your current plan or upgrade to a higher tier
          </CardDescription>
        </CardHeader>
        <CardContent>
          <RadioGroup value={selectedPlanID} onValueChange={setSelectedPlanID}>
            <div className="space-y-3">
              {/* Current plan */}
              <Card
                className={`cursor-pointer transition-colors ${
                  selectedPlanID === renewalData.current_plan.id
                    ? "border-primary bg-primary/5"
                    : "hover:border-primary/50"
                }`}
                onClick={() => setSelectedPlanID(renewalData.current_plan.id)}
              >
                <CardContent className="p-4">
                  <div className="flex items-start gap-4">
                    <RadioGroupItem
                      value={renewalData.current_plan.id}
                      id={renewalData.current_plan.id}
                      className="mt-1"
                    />
                    <div className="flex-1">
                      <Label htmlFor={renewalData.current_plan.id} className="cursor-pointer">
                        <div className="font-semibold">{renewalData.current_plan.name}</div>
                        <div className="text-sm text-muted-foreground mt-1">
                          Renew for {renewalData.current_plan.renewal_days || 365} days
                        </div>
                      </Label>
                    </div>
                  </div>
                </CardContent>
              </Card>

              {/* Task 15.3: Upgrade options (if available plans provided) */}
              {renewalData.available_plans?.map((plan) => (
                <Card
                  key={plan.id}
                  className={`cursor-pointer transition-colors ${
                    selectedPlanID === plan.id
                      ? "border-primary bg-primary/5"
                      : "hover:border-primary/50"
                  }`}
                  onClick={() => setSelectedPlanID(plan.id)}
                >
                  <CardContent className="p-4">
                    <div className="flex items-start gap-4">
                      <RadioGroupItem value={plan.id} id={plan.id} className="mt-1" />
                      <div className="flex-1">
                        <Label htmlFor={plan.id} className="cursor-pointer">
                          <div className="flex items-center gap-2">
                            <span className="font-semibold">{plan.name}</span>
                            <ArrowUp className="h-4 w-4 text-primary" />
                            <span className="text-xs text-primary font-medium">UPGRADE</span>
                          </div>
                          <div className="text-sm text-muted-foreground mt-1">
                            Upgrade and renew for {plan.renewal_days || 365} days
                          </div>
                        </Label>
                      </div>
                    </div>
                  </CardContent>
                </Card>
              ))}
            </div>
          </RadioGroup>

          {isUpgrade && (
            <div className="mt-4 p-3 bg-blue-50 rounded-md text-sm text-blue-900 dark:bg-blue-900/20 dark:text-blue-200">
              You're upgrading to a higher tier plan. Your license will be extended from the renewal date.
            </div>
          )}

          <Button
            className="w-full mt-6"
            size="lg"
            onClick={() => setShowPaymentSelector(true)}
            disabled={!selectedPlanID}
          >
            <RefreshCw className="mr-2 h-4 w-4" />
            Continue to Payment
          </Button>
        </CardContent>
      </Card>

      {/* Payment method selector */}
      {showPaymentSelector && selectedPlanID && renewalData.current_plan && (
        <PaymentMethodSelector
          plan={renewalData.current_plan}
          onCheckout={(provider) => checkoutMutation.mutate(provider)}
          loading={checkoutMutation.isPending}
        />
      )}
    </div>
  )
}
