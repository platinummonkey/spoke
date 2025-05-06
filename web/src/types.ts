export interface Module {
  name: string;
  description: string;
  versions: string[];
}

export interface Version {
  version: string;
  files: ProtoFile[];
  dependencies: string[];
}

export interface ProtoFile {
  path: string;
  content: string;
  messages: Message[];
  enums: Enum[];
  services: Service[];
}

export interface Message {
  name: string;
  fields: Field[];
  nestedMessages: Message[];
  nestedEnums: Enum[];
}

export interface Field {
  name: string;
  type: string;
  number: number;
  label: 'optional' | 'repeated' | 'required';
  options: Record<string, any>;
}

export interface Enum {
  name: string;
  values: EnumValue[];
}

export interface EnumValue {
  name: string;
  number: number;
}

export interface Service {
  name: string;
  methods: RpcMethod[];
}

export interface RpcMethod {
  name: string;
  inputType: string;
  outputType: string;
  clientStreaming: boolean;
  serverStreaming: boolean;
} 