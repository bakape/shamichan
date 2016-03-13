interface FileReaderOnloadEvent extends Event {
	target: FileReaderOnloadTarget
}

interface FileReaderOnloadTarget extends EventTarget {
	result: any
}

interface FileReader {
	onload: (event: FileReaderOnloadEvent) => void
}
