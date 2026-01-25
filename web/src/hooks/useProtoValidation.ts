import { useState, useCallback } from 'react';
import { Message, Field } from '../types';

export interface ValidationError {
  field: string;
  message: string;
  path: string;
}

interface ValidationResult {
  valid: boolean;
  errors: ValidationError[];
}

/**
 * Hook to validate JSON against protobuf message schema
 */
export const useProtoValidation = () => {
  const [errors, setErrors] = useState<ValidationError[]>([]);

  const validateField = useCallback((
    fieldName: string,
    fieldDef: Field,
    value: any,
    path: string = ''
  ): ValidationError[] => {
    const currentPath = path ? `${path}.${fieldName}` : fieldName;
    const errors: ValidationError[] = [];

    // Check if value is missing (only for non-repeated, non-optional fields)
    if (value === undefined || value === null) {
      // In proto3, all fields are optional by default
      // Only validate if explicitly required (which doesn't exist in proto3)
      return errors;
    }

    // Validate repeated fields
    if (fieldDef.label === 'repeated') {
      if (!Array.isArray(value)) {
        errors.push({
          field: fieldName,
          path: currentPath,
          message: `Field '${fieldName}' should be an array (repeated field)`,
        });
        return errors;
      }

      // Validate each element
      value.forEach((item, index) => {
        const itemErrors = validateFieldValue(
          fieldName,
          fieldDef.type,
          item,
          `${currentPath}[${index}]`
        );
        errors.push(...itemErrors);
      });

      return errors;
    }

    // Validate single field value
    const valueErrors = validateFieldValue(fieldName, fieldDef.type, value, currentPath);
    errors.push(...valueErrors);

    return errors;
  }, []);

  const validateFieldValue = (
    fieldName: string,
    fieldType: string,
    value: any,
    path: string
  ): ValidationError[] => {
    const errors: ValidationError[] = [];

    // Scalar type validation
    switch (fieldType) {
      case 'string':
      case 'bytes':
        if (typeof value !== 'string') {
          errors.push({
            field: fieldName,
            path,
            message: `Field '${fieldName}' should be a string, got ${typeof value}`,
          });
        }
        break;

      case 'int32':
      case 'int64':
      case 'uint32':
      case 'uint64':
      case 'sint32':
      case 'sint64':
      case 'fixed32':
      case 'fixed64':
      case 'sfixed32':
      case 'sfixed64':
        if (typeof value !== 'number' || !Number.isInteger(value)) {
          errors.push({
            field: fieldName,
            path,
            message: `Field '${fieldName}' should be an integer, got ${typeof value}`,
          });
        }
        break;

      case 'float':
      case 'double':
        if (typeof value !== 'number') {
          errors.push({
            field: fieldName,
            path,
            message: `Field '${fieldName}' should be a number, got ${typeof value}`,
          });
        }
        break;

      case 'bool':
        if (typeof value !== 'boolean') {
          errors.push({
            field: fieldName,
            path,
            message: `Field '${fieldName}' should be a boolean, got ${typeof value}`,
          });
        }
        break;

      default:
        // Message type - should be an object
        if (fieldType[0] === fieldType[0].toUpperCase()) {
          if (typeof value !== 'object' || value === null || Array.isArray(value)) {
            errors.push({
              field: fieldName,
              path,
              message: `Field '${fieldName}' should be an object (message type ${fieldType})`,
            });
          }
        }
        break;
    }

    return errors;
  };

  const validate = useCallback((message: Message, data: any): ValidationResult => {
    if (typeof data !== 'object' || data === null) {
      setErrors([{
        field: 'root',
        path: '',
        message: 'Data should be a JSON object',
      }]);
      return { valid: false, errors: [{
        field: 'root',
        path: '',
        message: 'Data should be a JSON object',
      }]};
    }

    const allErrors: ValidationError[] = [];

    // Validate each field in the message
    for (const field of message.fields) {
      const value = data[field.name];
      const fieldErrors = validateField(field.name, field, value);
      allErrors.push(...fieldErrors);
    }

    // Check for unknown fields
    const knownFields = new Set(message.fields.map(f => f.name));
    for (const key of Object.keys(data)) {
      if (!knownFields.has(key)) {
        allErrors.push({
          field: key,
          path: key,
          message: `Unknown field '${key}' not defined in message ${message.name}`,
        });
      }
    }

    setErrors(allErrors);
    return {
      valid: allErrors.length === 0,
      errors: allErrors,
    };
  }, [validateField]);

  const clearErrors = useCallback(() => {
    setErrors([]);
  }, []);

  return {
    validate,
    errors,
    clearErrors,
  };
};
