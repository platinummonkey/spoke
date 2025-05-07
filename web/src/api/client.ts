import axios from 'axios';
import { Module, Version, ProtoFile, Message, Enum, Service } from '../types';
import * as protobuf from 'protobufjs';

const api = axios.create({
  baseURL: '/api',
});

const handleResponse = <T>(response: any): T => {
  if (!response.data) {
    throw new Error('No data received from server');
  }
  return response.data as T;
};

// Helper function to parse protobuf content using protobufjs
const parseProtoFile = async (content: string): Promise<{ messages: Message[]; enums: Enum[]; services: Service[] }> => {
  const messages: Message[] = [];
  const enums: Enum[] = [];
  const services: Service[] = [];

  try {
    // Parse the content directly
    console.log('Parsing content:', content);
    const parsed = protobuf.parse(content);
    console.log('Parsed content:', parsed);

    // Process the parsed content
    if (parsed.root && parsed.root.nested) {
      // Get the package namespace
      const packageName = parsed.package;
      if (packageName && parsed.root.nested[packageName]?.nested) {
        const packageNs = parsed.root.nested[packageName].nested;
        
        // Process each type in the package
        Object.entries(packageNs).forEach(([name, obj]) => {
          // Process messages
          if (obj.fields) {
            const message: Message = {
              name,
              fields: Object.entries(obj.fields).map(([fieldName, field]) => ({
                name: fieldName,
                type: field.type,
                number: field.id,
                label: field.repeated ? 'repeated' : 'optional',
                options: field.options || {},
              })),
              nestedMessages: [],
              nestedEnums: [],
            };
            messages.push(message);
          }
          
          // Process enums
          if (obj.values) {
            const enumType: Enum = {
              name,
              values: Object.entries(obj.values).map(([valueName, value]) => ({
                name: valueName,
                number: value,
              })),
            };
            enums.push(enumType);
          }
          
          // Process services
          if (obj.methods) {
            const service: Service = {
              name,
              methods: Object.entries(obj.methods).map(([methodName, method]) => ({
                name: methodName,
                inputType: method.requestType,
                outputType: method.responseType,
                clientStreaming: method.requestStream,
                serverStreaming: method.responseStream,
              })),
            };
            services.push(service);
          }
        });
      }
    }

    console.log('Processed proto file:', { messages, enums, services });
    return { messages, enums, services };
  } catch (error) {
    console.error('Error parsing proto file:', error);
    return { messages: [], enums: [], services: [] };
  }
};

const processProtoFiles = async (files: any[]): Promise<ProtoFile[]> => {
  const processedFiles: ProtoFile[] = [];
  
  for (const file of files) {
    try {
      const parsed = await parseProtoFile(file.content);
      processedFiles.push({
        ...file,
        ...parsed,
      });
    } catch (error) {
      console.error(`Error processing file ${file.path}:`, error);
      processedFiles.push({
        ...file,
        messages: [],
        enums: [],
        services: [],
      });
    }
  }
  
  return processedFiles;
};

export const getModules = async (): Promise<Module[]> => {
  const response = await api.get('/modules');
  const data = handleResponse<Module[]>(response);
  return data.map(module => ({
    ...module,
    versions: module.versions || [],
  }));
};

export const getModule = async (name: string): Promise<Module> => {
  const response = await api.get(`/modules/${name}`);
  const data = handleResponse<Module>(response);
  
  // Ensure versions exists and is an array
  const versions = Array.isArray(data.versions) ? data.versions : [];
  
  // Process proto files for each version and sort by newest first
  const processedVersions = await Promise.all(versions.map(async version => ({
    ...version,
    files: await processProtoFiles(version.files || []),
  })));

  // Sort versions by created_at timestamp, newest first
  const sortedVersions = processedVersions.sort((a, b) => 
    new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
  );
  
  return {
    ...data,
    versions: sortedVersions,
  };
};

interface FileResponse {
  content: string;
}

export const getFile = async (moduleName: string, version: string, path: string): Promise<FileResponse> => {
  const response = await api.get(`/modules/${moduleName}/versions/${version}/files/${path}`);
  return handleResponse<FileResponse>(response);
};

export const getVersion = async (moduleName: string, version: string): Promise<Version> => {
  const response = await api.get(`/modules/${moduleName}/versions/${version}`);
  const data = handleResponse<Version>(response);
  return {
    ...data,
    files: await processProtoFiles(data.files || []),
    dependencies: data.dependencies || [],
  };
};

export const createModule = async (module: Omit<Module, 'versions'>): Promise<Module> => {
  const response = await api.post('/modules', module);
  const data = handleResponse<Module>(response);
  return {
    ...data,
    versions: data.versions || [],
  };
};

export const createVersion = async (moduleName: string, version: Omit<Version, 'moduleName'>): Promise<Version> => {
  const response = await api.post(`/modules/${moduleName}/versions`, version);
  const data = handleResponse<Version>(response);
  return {
    ...data,
    files: await processProtoFiles(data.files || []),
    dependencies: data.dependencies || [],
  };
}; 