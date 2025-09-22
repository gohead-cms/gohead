interface Attribute {

    type: string;
    required?: boolean;
    enum?: string[];
    target?: string;
}

export interface Schema {
    attributes: Record<string, any>;
    collectionName?: string;
    info?: {
        displayName?: string;
        pluralName?: string;
        singularName?: string;
    };
}