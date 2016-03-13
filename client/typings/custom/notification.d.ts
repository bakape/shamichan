interface Notification {
	permission: string
	requestPermission(): void
}

interface Window {
	Notification: Notification
}

declare var Notification: typeof window.Notification
