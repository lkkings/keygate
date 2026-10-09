import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { AlertCircle, CheckCircle, Copy, CreditCard, Eye, EyeOff, RefreshCw } from "lucide-react"
import { useState } from "react"
import { showToast } from "@/components/toast"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Switch } from "@/components/ui/switch"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Textarea } from "@/components/ui/textarea"
import { type TranslationKeys, useI18n } from "@/i18n"

type Translate = (key: TranslationKeys, params?: Record<string, string | number>) => string

export function PaymentSettingsPage() {
  const { t } = useI18n()
  const queryClient = useQueryClient()
  const { data, isLoading } = useQuery({
    queryKey: ["payment-settings"],
    queryFn: async () => {
      const response = await fetch("/api/v1/admin/settings")
      return response.json()
    },
  })

  if (isLoading) {
    return (
      <div className="flex items-center justify-center p-8">
        <RefreshCw className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold">{t("paymentSettings.title")}</h1>
        <p className="text-muted-foreground mt-2">{t("paymentSettings.subtitle")}</p>
      </div>

      {/* Task 10.10: Responsive tabs - stack on mobile, side-by-side on desktop */}
      <Tabs defaultValue="stripe" className="space-y-6">
        <TabsList className="grid w-full grid-cols-2 lg:grid-cols-4">
          <TabsTrigger value="stripe">{t("paymentSettings.provider.stripe")}</TabsTrigger>
          <TabsTrigger value="alipay">{t("paymentSettings.provider.alipay")}</TabsTrigger>
          <TabsTrigger value="wechat">{t("paymentSettings.provider.wechat")}</TabsTrigger>
          <TabsTrigger value="epay">{t("paymentSettings.provider.epay")}</TabsTrigger>
        </TabsList>

        <TabsContent value="stripe">
          <StripeConfigCard
            settings={data?.providers?.stripe || {}}
            webhookUrl={data?.webhook_urls?.stripe || ""}
            onUpdate={() => queryClient.invalidateQueries({ queryKey: ["payment-settings"] })}
          />
        </TabsContent>

        <TabsContent value="alipay">
          <AlipayConfigCard
            settings={data?.providers?.alipay || {}}
            webhookUrl={data?.webhook_urls?.alipay || ""}
            onUpdate={() => queryClient.invalidateQueries({ queryKey: ["payment-settings"] })}
          />
        </TabsContent>

        <TabsContent value="wechat">
          <WechatConfigCard
            settings={data?.providers?.wechat || {}}
            webhookUrl={data?.webhook_urls?.wechat || ""}
            onUpdate={() => queryClient.invalidateQueries({ queryKey: ["payment-settings"] })}
          />
        </TabsContent>

        <TabsContent value="epay">
          <EpayConfigCard
            settings={data?.providers?.epay || {}}
            webhookUrl={data?.webhook_urls?.epay || ""}
            onUpdate={() => queryClient.invalidateQueries({ queryKey: ["payment-settings"] })}
          />
        </TabsContent>
      </Tabs>
    </div>
  )
}

// Shared save mutation for a provider card: PUTs the provider's settings
// and reports the outcome in the current language.
function useSaveProvider(provider: string, providerName: string, onUpdate: () => void) {
  const { t } = useI18n()
  return useMutation({
    mutationFn: async (data: Record<string, string>) => {
      const response = await fetch(`/api/v1/admin/payment-providers/${provider}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(data),
      })
      if (!response.ok) throw new Error(t("settings.saveError"))
      return response.json()
    },
    onSuccess: () => {
      showToast(t("paymentSettings.saved", { provider: providerName }), "success")
      onUpdate()
    },
    onError: (error: Error) => {
      showToast(t("paymentSettings.saveFailed", { error: error.message }), "error")
    },
  })
}

// Stripe Configuration Card (Task 10.2)
function StripeConfigCard({
  settings,
  webhookUrl,
  onUpdate,
}: {
  settings: Record<string, string>
  webhookUrl: string
  onUpdate: () => void
}) {
  const { t } = useI18n()
  const name = t("paymentSettings.provider.stripe")
  const [formData, setFormData] = useState({
    enabled: settings.enabled === "true",
    secret_key: settings.secret_key || "",
    publishable_key: settings.publishable_key || "",
    webhook_secret: settings.webhook_secret || "",
  })

  const updateMutation = useSaveProvider("stripe", name, onUpdate)

  const testMutation = useMutation({
    mutationFn: async () => {
      const response = await fetch("/api/v1/admin/payment-providers/stripe/test", {
        method: "POST",
      })
      if (!response.ok) throw new Error(response.statusText || String(response.status))
      return response.json()
    },
    onSuccess: (data: { status?: string }) => {
      if (data.status === "not_implemented") {
        showToast(t("paymentSettings.testNotImplemented"), "success")
      } else {
        showToast(t("paymentSettings.testSuccess"), "success")
      }
    },
    onError: (error: Error) => {
      showToast(t("paymentSettings.testFailed", { error: error.message }), "error")
    },
  })

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    updateMutation.mutate({
      enabled: formData.enabled ? "true" : "false",
      secret_key: formData.secret_key,
      publishable_key: formData.publishable_key,
      webhook_secret: formData.webhook_secret,
    })
  }

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <CreditCard className="h-5 w-5" />
            <CardTitle>{t("paymentSettings.configTitle", { provider: name })}</CardTitle>
          </div>
          {formData.enabled && <CheckCircle className="h-5 w-5 text-green-600" />}
        </div>
        <CardDescription>{t("paymentSettings.stripeDesc")}</CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="flex items-center justify-between">
            <Label htmlFor="stripe-enabled">{t("paymentSettings.enable", { provider: name })}</Label>
            <Switch
              id="stripe-enabled"
              checked={formData.enabled}
              onCheckedChange={(checked: boolean) => setFormData({ ...formData, enabled: checked })}
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="stripe-secret-key">{t("paymentSettings.secretKey")}</Label>
            <PasswordInput
              id="stripe-secret-key"
              placeholder="sk_live_..."
              value={formData.secret_key}
              onChange={(value) => setFormData({ ...formData, secret_key: value })}
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="stripe-publishable-key">{t("paymentSettings.publishableKey")}</Label>
            <Input
              id="stripe-publishable-key"
              placeholder="pk_live_..."
              value={formData.publishable_key}
              onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) =>
                setFormData({ ...formData, publishable_key: e.target.value })
              }
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="stripe-webhook-secret">{t("paymentSettings.webhookSecret")}</Label>
            <PasswordInput
              id="stripe-webhook-secret"
              placeholder="whsec_..."
              value={formData.webhook_secret}
              onChange={(value) => setFormData({ ...formData, webhook_secret: value })}
            />
          </div>

          <WebhookUrlDisplay url={webhookUrl} />

          <div className="flex gap-2">
            <Button type="submit" disabled={updateMutation.isPending}>
              {updateMutation.isPending ? t("paymentSettings.saving") : t("paymentSettings.save")}
            </Button>
            <Button
              type="button"
              variant="outline"
              onClick={() => testMutation.mutate()}
              disabled={testMutation.isPending || !formData.enabled}
            >
              {testMutation.isPending ? t("paymentSettings.testing") : t("paymentSettings.testConnection")}
            </Button>
          </div>
        </form>
      </CardContent>
    </Card>
  )
}

// Alipay Configuration Card (Task 10.3)
function AlipayConfigCard({
  settings,
  webhookUrl,
  onUpdate,
}: {
  settings: Record<string, string>
  webhookUrl: string
  onUpdate: () => void
}) {
  const { t } = useI18n()
  const name = t("paymentSettings.provider.alipay")
  const [formData, setFormData] = useState({
    enabled: settings.enabled === "true",
    app_id: settings.app_id || "",
    private_key: settings.private_key || "",
    public_key: settings.public_key || "",
    sandbox_mode: settings.sandbox_mode === "true",
  })

  const [errors, setErrors] = useState<Record<string, string>>({})

  const updateMutation = useSaveProvider("alipay", name, onUpdate)

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()

    // Validate app_id
    const appIdError = validateAlipayAppId(formData.app_id, t)
    if (appIdError) {
      setErrors({ app_id: appIdError })
      return
    }

    setErrors({})
    updateMutation.mutate({
      enabled: formData.enabled ? "true" : "false",
      app_id: formData.app_id,
      private_key: formData.private_key,
      public_key: formData.public_key,
      sandbox_mode: formData.sandbox_mode ? "true" : "false",
    })
  }

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <CreditCard className="h-5 w-5" />
            <CardTitle>{t("paymentSettings.configTitle", { provider: name })}</CardTitle>
          </div>
          {formData.enabled && <CheckCircle className="h-5 w-5 text-green-600" />}
        </div>
        <CardDescription>{t("paymentSettings.alipayDesc")}</CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="flex items-center justify-between">
            <Label htmlFor="alipay-enabled">{t("paymentSettings.enable", { provider: name })}</Label>
            <Switch
              id="alipay-enabled"
              checked={formData.enabled}
              onCheckedChange={(checked: boolean) => setFormData({ ...formData, enabled: checked })}
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="alipay-app-id">{t("paymentSettings.appId")}</Label>
            <Input
              id="alipay-app-id"
              placeholder={t("paymentSettings.alipayAppIdPlaceholder")}
              value={formData.app_id}
              onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
                setFormData({ ...formData, app_id: e.target.value })
                setErrors({ ...errors, app_id: "" })
              }}
              className={errors.app_id ? "border-red-500" : ""}
            />
            {errors.app_id && <p className="text-xs text-red-500">{errors.app_id}</p>}
          </div>

          <div className="space-y-2">
            <Label htmlFor="alipay-private-key">{t("paymentSettings.privateKey")}</Label>
            <Textarea
              id="alipay-private-key"
              placeholder="-----BEGIN PRIVATE KEY-----&#10;...&#10;-----END PRIVATE KEY-----"
              className="font-mono text-sm"
              rows={6}
              value={formData.private_key}
              onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) =>
                setFormData({ ...formData, private_key: e.target.value })
              }
            />
            <p className="text-xs text-muted-foreground">{t("paymentSettings.privateKeyDesc")}</p>
          </div>

          <div className="space-y-2">
            <Label htmlFor="alipay-public-key">{t("paymentSettings.alipayPublicKey")}</Label>
            <Textarea
              id="alipay-public-key"
              placeholder="-----BEGIN PUBLIC KEY-----&#10;...&#10;-----END PUBLIC KEY-----"
              className="font-mono text-sm"
              rows={6}
              value={formData.public_key}
              onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) =>
                setFormData({ ...formData, public_key: e.target.value })
              }
            />
            <p className="text-xs text-muted-foreground">{t("paymentSettings.alipayPublicKeyDesc")}</p>
          </div>

          <div className="flex items-center justify-between">
            <div>
              <Label htmlFor="alipay-sandbox">{t("paymentSettings.sandboxMode")}</Label>
              <p className="text-xs text-muted-foreground">{t("paymentSettings.alipaySandboxDesc")}</p>
            </div>
            <Switch
              id="alipay-sandbox"
              checked={formData.sandbox_mode}
              onCheckedChange={(checked: boolean) => setFormData({ ...formData, sandbox_mode: checked })}
            />
          </div>

          <WebhookUrlDisplay url={webhookUrl} />

          <Button type="submit" disabled={updateMutation.isPending}>
            {updateMutation.isPending ? t("paymentSettings.saving") : t("paymentSettings.save")}
          </Button>
        </form>
      </CardContent>
    </Card>
  )
}

// WeChat Pay Configuration Card (Task 10.4)
function WechatConfigCard({
  settings,
  webhookUrl,
  onUpdate,
}: {
  settings: Record<string, string>
  webhookUrl: string
  onUpdate: () => void
}) {
  const { t } = useI18n()
  const name = t("paymentSettings.provider.wechat")
  const [formData, setFormData] = useState({
    enabled: settings.enabled === "true",
    app_id: settings.app_id || "",
    merchant_id: settings.merchant_id || "",
    api_key: settings.api_key || "",
    cert_file: settings.cert_file || "",
    key_file: settings.key_file || "",
    sandbox_mode: settings.sandbox_mode === "true",
  })

  const [errors, setErrors] = useState<Record<string, string>>({})

  const updateMutation = useSaveProvider("wechat", name, onUpdate)

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()

    // Validate app_id
    const appIdError = validateWechatAppId(formData.app_id, t)
    if (appIdError) {
      setErrors({ app_id: appIdError })
      return
    }

    setErrors({})
    updateMutation.mutate({
      enabled: formData.enabled ? "true" : "false",
      app_id: formData.app_id,
      merchant_id: formData.merchant_id,
      api_key: formData.api_key,
      cert_file: formData.cert_file,
      key_file: formData.key_file,
      sandbox_mode: formData.sandbox_mode ? "true" : "false",
    })
  }

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <CreditCard className="h-5 w-5" />
            <CardTitle>{t("paymentSettings.configTitle", { provider: name })}</CardTitle>
          </div>
          {formData.enabled && <CheckCircle className="h-5 w-5 text-green-600" />}
        </div>
        <CardDescription>{t("paymentSettings.wechatDesc")}</CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="flex items-center justify-between">
            <Label htmlFor="wechat-enabled">{t("paymentSettings.enable", { provider: name })}</Label>
            <Switch
              id="wechat-enabled"
              checked={formData.enabled}
              onCheckedChange={(checked: boolean) => setFormData({ ...formData, enabled: checked })}
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="wechat-app-id">{t("paymentSettings.appId")}</Label>
            <Input
              id="wechat-app-id"
              placeholder={t("paymentSettings.wechatAppIdPlaceholder")}
              value={formData.app_id}
              onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
                setFormData({ ...formData, app_id: e.target.value })
                setErrors({ ...errors, app_id: "" })
              }}
              className={errors.app_id ? "border-red-500" : ""}
            />
            {errors.app_id && <p className="text-xs text-red-500">{errors.app_id}</p>}
          </div>

          <div className="space-y-2">
            <Label htmlFor="wechat-merchant-id">{t("paymentSettings.merchantId")}</Label>
            <Input
              id="wechat-merchant-id"
              placeholder="1234567890"
              value={formData.merchant_id}
              onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) =>
                setFormData({ ...formData, merchant_id: e.target.value })
              }
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="wechat-api-key">{t("paymentSettings.apiKey")}</Label>
            <PasswordInput
              id="wechat-api-key"
              placeholder={t("paymentSettings.wechatApiKeyPlaceholder")}
              value={formData.api_key}
              onChange={(value) => setFormData({ ...formData, api_key: value })}
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="wechat-cert-file">{t("paymentSettings.certFile")}</Label>
            <Input
              id="wechat-cert-file"
              placeholder="/path/to/apiclient_cert.pem"
              value={formData.cert_file}
              onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) =>
                setFormData({ ...formData, cert_file: e.target.value })
              }
            />
            <p className="text-xs text-muted-foreground">{t("paymentSettings.certFileDesc")}</p>
          </div>

          <div className="space-y-2">
            <Label htmlFor="wechat-key-file">{t("paymentSettings.keyFile")}</Label>
            <Input
              id="wechat-key-file"
              placeholder="/path/to/apiclient_key.pem"
              value={formData.key_file}
              onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) =>
                setFormData({ ...formData, key_file: e.target.value })
              }
            />
            <p className="text-xs text-muted-foreground">{t("paymentSettings.keyFileDesc")}</p>
          </div>

          <div className="flex items-center justify-between">
            <div>
              <Label htmlFor="wechat-sandbox">{t("paymentSettings.sandboxMode")}</Label>
              <p className="text-xs text-muted-foreground">{t("paymentSettings.wechatSandboxDesc")}</p>
            </div>
            <Switch
              id="wechat-sandbox"
              checked={formData.sandbox_mode}
              onCheckedChange={(checked: boolean) => setFormData({ ...formData, sandbox_mode: checked })}
            />
          </div>

          <div className="rounded-md bg-blue-50 p-4 text-sm text-blue-900 dark:bg-blue-900/20 dark:text-blue-200">
            <div className="flex gap-2">
              <AlertCircle className="h-5 w-5 flex-shrink-0" />
              <div>
                <p className="font-medium">{t("paymentSettings.wechatModeTitle")}</p>
                <p className="mt-1 text-xs">{t("paymentSettings.wechatModeDesc")}</p>
              </div>
            </div>
          </div>

          <WebhookUrlDisplay url={webhookUrl} />

          <Button type="submit" disabled={updateMutation.isPending}>
            {updateMutation.isPending ? t("paymentSettings.saving") : t("paymentSettings.save")}
          </Button>
        </form>
      </CardContent>
    </Card>
  )
}

// ePay Configuration Card (Task 10.5)
function EpayConfigCard({
  settings,
  webhookUrl,
  onUpdate,
}: {
  settings: Record<string, string>
  webhookUrl: string
  onUpdate: () => void
}) {
  const { t } = useI18n()
  const name = t("paymentSettings.provider.epay")
  const [formData, setFormData] = useState({
    enabled: settings.enabled === "true",
    merchant_id: settings.merchant_id || "",
    api_key: settings.api_key || "",
    gateway_url: settings.gateway_url || "",
    sandbox_mode: settings.sandbox_mode === "true",
    signature_algorithm: settings.signature_algorithm || "MD5",
  })

  const updateMutation = useSaveProvider("epay", name, onUpdate)

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    updateMutation.mutate({
      enabled: formData.enabled ? "true" : "false",
      merchant_id: formData.merchant_id,
      api_key: formData.api_key,
      gateway_url: formData.gateway_url,
      sandbox_mode: formData.sandbox_mode ? "true" : "false",
      signature_algorithm: formData.signature_algorithm,
    })
  }

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <CreditCard className="h-5 w-5" />
            <CardTitle>{t("paymentSettings.configTitle", { provider: name })}</CardTitle>
          </div>
          {formData.enabled && <CheckCircle className="h-5 w-5 text-green-600" />}
        </div>
        <CardDescription>{t("paymentSettings.epayDesc")}</CardDescription>
      </CardHeader>
      <CardContent>
        <div className="rounded-md bg-yellow-50 p-4 text-sm text-yellow-900 dark:bg-yellow-900/20 dark:text-yellow-200 mb-4">
          <div className="flex gap-2">
            <AlertCircle className="h-5 w-5 flex-shrink-0" />
            <div>
              <p className="font-medium">{t("paymentSettings.epayStubTitle")}</p>
              <p className="mt-1 text-xs">{t("paymentSettings.epayStubDesc")}</p>
            </div>
          </div>
        </div>

        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="flex items-center justify-between">
            <Label htmlFor="epay-enabled">{t("paymentSettings.enable", { provider: name })}</Label>
            <Switch
              id="epay-enabled"
              checked={formData.enabled}
              onCheckedChange={(checked: boolean) => setFormData({ ...formData, enabled: checked })}
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="epay-merchant-id">{t("paymentSettings.merchantId")}</Label>
            <Input
              id="epay-merchant-id"
              placeholder={t("paymentSettings.merchantIdPlaceholder")}
              value={formData.merchant_id}
              onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) =>
                setFormData({ ...formData, merchant_id: e.target.value })
              }
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="epay-api-key">{t("paymentSettings.apiKey")}</Label>
            <PasswordInput
              id="epay-api-key"
              placeholder={t("paymentSettings.apiKeyPlaceholder")}
              value={formData.api_key}
              onChange={(value) => setFormData({ ...formData, api_key: value })}
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="epay-gateway">{t("paymentSettings.gatewayUrl")}</Label>
            <Input
              id="epay-gateway"
              placeholder="https://api.epay.example.com"
              value={formData.gateway_url}
              onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) =>
                setFormData({ ...formData, gateway_url: e.target.value })
              }
            />
            <p className="text-xs text-muted-foreground">{t("paymentSettings.gatewayUrlDesc")}</p>
          </div>

          <div className="space-y-2">
            <Label htmlFor="epay-signature">{t("paymentSettings.signatureAlgorithm")}</Label>
            <Select
              value={formData.signature_algorithm}
              onValueChange={(value) => setFormData({ ...formData, signature_algorithm: value })}
            >
              <SelectTrigger id="epay-signature">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="MD5">MD5</SelectItem>
                <SelectItem value="HMAC-SHA256">HMAC-SHA256</SelectItem>
              </SelectContent>
            </Select>
          </div>

          <div className="flex items-center justify-between">
            <div>
              <Label htmlFor="epay-sandbox">{t("paymentSettings.sandboxMode")}</Label>
              <p className="text-xs text-muted-foreground">{t("paymentSettings.epaySandboxDesc")}</p>
            </div>
            <Switch
              id="epay-sandbox"
              checked={formData.sandbox_mode}
              onCheckedChange={(checked: boolean) => setFormData({ ...formData, sandbox_mode: checked })}
            />
          </div>

          <WebhookUrlDisplay url={webhookUrl} />

          <Button type="submit" disabled={updateMutation.isPending}>
            {updateMutation.isPending ? t("paymentSettings.saving") : t("paymentSettings.save")}
          </Button>
        </form>
      </CardContent>
    </Card>
  )
}

// Reusable webhook URL display component
function WebhookUrlDisplay({ url }: { url: string }) {
  const { t } = useI18n()
  const [copied, setCopied] = useState(false)

  const copyToClipboard = () => {
    navigator.clipboard.writeText(url)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  return (
    <div className="space-y-2">
      <Label>{t("paymentSettings.webhookUrl")}</Label>
      <div className="flex gap-2">
        <Input value={url} readOnly className="font-mono text-sm" />
        <Button
          type="button"
          variant="outline"
          size="icon"
          onClick={copyToClipboard}
          aria-label={t("paymentSettings.copyWebhookUrl")}
        >
          {copied ? <CheckCircle className="h-4 w-4" /> : <Copy className="h-4 w-4" />}
        </Button>
      </div>
      <p className="text-xs text-muted-foreground">{t("paymentSettings.webhookUrlDesc")}</p>
    </div>
  )
}

// Password input with show/hide toggle (Task 10.9)
function PasswordInput({
  id,
  value,
  onChange,
  placeholder,
  error,
}: {
  id: string
  value: string
  onChange: (value: string) => void
  placeholder?: string
  error?: string
}) {
  const { t } = useI18n()
  const [showPassword, setShowPassword] = useState(false)

  return (
    <div className="space-y-1">
      <div className="relative">
        <Input
          id={id}
          type={showPassword ? "text" : "password"}
          placeholder={placeholder}
          value={value}
          onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => onChange(e.target.value)}
          className={error ? "border-red-500" : ""}
        />
        <Button
          type="button"
          variant="ghost"
          size="icon"
          className="absolute right-0 top-0 h-full px-3 hover:bg-transparent"
          onClick={() => setShowPassword(!showPassword)}
          aria-label={showPassword ? t("paymentSettings.hideSecret") : t("paymentSettings.showSecret")}
        >
          {showPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
        </Button>
      </div>
      {error && <p className="text-xs text-red-500">{error}</p>}
    </div>
  )
}

// Validation helpers (Task 10.8)
function validateAlipayAppId(appId: string, t: Translate): string | undefined {
  if (!appId) return undefined
  if (!/^\d{16}$/.test(appId)) {
    return t("paymentSettings.alipayAppIdInvalid")
  }
  return undefined
}

function validateWechatAppId(appId: string, t: Translate): string | undefined {
  if (!appId) return undefined
  if (!appId.startsWith("wx")) {
    return t("paymentSettings.wechatAppIdInvalid")
  }
  return undefined
}
