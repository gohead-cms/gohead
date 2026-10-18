interface Attribute {

    type: string;
    required?: boolean;
    enum?: string[];
    target?: string;
}

export interface Schema {
    attributes: Record<string, any>;
    collectionName?: string;
    options?: {
        draftAndPublish?: boolean;
    };
    info?: {
        displayName?: string;
        pluralName?: string;
        singularName?: string;
    };
}