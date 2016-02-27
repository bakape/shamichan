interface Document {
	query(selector: string): Element;
	queryAll(selector: string): Element[];
}

interface Element {
	query(selector: string): Element;
	queryAll(selector: string): Element[];
}
