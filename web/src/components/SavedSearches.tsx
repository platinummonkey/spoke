import React, { useState } from 'react';
import {
  Box,
  VStack,
  HStack,
  Text,
  Button,
  IconButton,
  Input,
  Textarea,
  Modal,
  ModalOverlay,
  ModalContent,
  ModalHeader,
  ModalBody,
  ModalFooter,
  ModalCloseButton,
  useDisclosure,
  Menu,
  MenuButton,
  MenuList,
  MenuItem,
  useToast,
  Tooltip,
  Divider,
} from '@chakra-ui/react';
import { AddIcon, DeleteIcon, EditIcon, SearchIcon, ChevronDownIcon } from '@chakra-ui/icons';
import { useSavedSearches, SavedSearch } from '../hooks/useSavedSearches';

interface SavedSearchesProps {
  onSelectSearch?: (query: string) => void;
}

export const SavedSearches: React.FC<SavedSearchesProps> = ({ onSelectSearch }) => {
  const { searches, loading, createSearch, updateSearch, deleteSearch } = useSavedSearches();
  const { isOpen, onOpen, onClose } = useDisclosure();
  const [editingSearch, setEditingSearch] = useState<SavedSearch | null>(null);
  const [formData, setFormData] = useState({ name: '', query: '', description: '' });
  const toast = useToast();

  const handleCreate = () => {
    setEditingSearch(null);
    setFormData({ name: '', query: '', description: '' });
    onOpen();
  };

  const handleEdit = (search: SavedSearch) => {
    setEditingSearch(search);
    setFormData({
      name: search.name,
      query: search.query,
      description: search.description || '',
    });
    onOpen();
  };

  const handleSave = async () => {
    try {
      if (editingSearch) {
        await updateSearch(editingSearch.id, formData);
        toast({
          title: 'Search updated',
          status: 'success',
          duration: 2000,
        });
      } else {
        await createSearch(formData);
        toast({
          title: 'Search saved',
          status: 'success',
          duration: 2000,
        });
      }
      onClose();
    } catch (err) {
      toast({
        title: 'Error',
        description: err instanceof Error ? err.message : 'Failed to save search',
        status: 'error',
        duration: 3000,
      });
    }
  };

  const handleDelete = async (id: number, name: string) => {
    if (window.confirm(`Delete saved search "${name}"?`)) {
      try {
        await deleteSearch(id);
        toast({
          title: 'Search deleted',
          status: 'success',
          duration: 2000,
        });
      } catch (err) {
        toast({
          title: 'Error',
          description: 'Failed to delete search',
          status: 'error',
          duration: 3000,
        });
      }
    }
  };

  const handleSelectSearch = (query: string) => {
    if (onSelectSearch) {
      onSelectSearch(query);
    }
  };

  if (loading) {
    return (
      <Box p={4}>
        <Text fontSize="sm" color="gray.600">
          Loading saved searches...
        </Text>
      </Box>
    );
  }

  return (
    <Box>
      <HStack justify="space-between" mb={4}>
        <Text fontWeight="bold" fontSize="md">
          Saved Searches
        </Text>
        <Tooltip label="Save new search">
          <IconButton
            aria-label="Save new search"
            icon={<AddIcon />}
            size="xs"
            onClick={handleCreate}
          />
        </Tooltip>
      </HStack>

      {searches.length === 0 ? (
        <Box p={4} bg="gray.50" borderRadius="md">
          <Text fontSize="sm" color="gray.600" textAlign="center">
            No saved searches yet
          </Text>
          <Button size="sm" mt={2} onClick={handleCreate} width="full">
            Create your first search
          </Button>
        </Box>
      ) : (
        <VStack align="stretch" spacing={2}>
          {searches.map((search) => (
            <Box
              key={search.id}
              p={3}
              bg="white"
              borderRadius="md"
              borderWidth={1}
              borderColor="gray.200"
              _hover={{ borderColor: 'blue.300', bg: 'blue.50' }}
              transition="all 0.2s"
            >
              <HStack justify="space-between" align="start">
                <VStack align="stretch" spacing={1} flex={1}>
                  <Text fontWeight="medium" fontSize="sm">
                    {search.name}
                  </Text>
                  <Text fontSize="xs" color="gray.600" noOfLines={1}>
                    {search.query}
                  </Text>
                  {search.description && (
                    <Text fontSize="xs" color="gray.500" noOfLines={2}>
                      {search.description}
                    </Text>
                  )}
                </VStack>
                <Menu>
                  <MenuButton
                    as={IconButton}
                    icon={<ChevronDownIcon />}
                    size="xs"
                    variant="ghost"
                    aria-label="Search options"
                  />
                  <MenuList fontSize="sm">
                    <MenuItem icon={<SearchIcon />} onClick={() => handleSelectSearch(search.query)}>
                      Execute Search
                    </MenuItem>
                    <MenuItem icon={<EditIcon />} onClick={() => handleEdit(search)}>
                      Edit
                    </MenuItem>
                    <Divider />
                    <MenuItem
                      icon={<DeleteIcon />}
                      onClick={() => handleDelete(search.id, search.name)}
                      color="red.600"
                    >
                      Delete
                    </MenuItem>
                  </MenuList>
                </Menu>
              </HStack>
            </Box>
          ))}
        </VStack>
      )}

      {/* Create/Edit Modal */}
      <Modal isOpen={isOpen} onClose={onClose}>
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>{editingSearch ? 'Edit' : 'Save'} Search</ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            <VStack spacing={4}>
              <Box width="full">
                <Text fontSize="sm" fontWeight="medium" mb={1}>
                  Name
                </Text>
                <Input
                  placeholder="e.g., User Messages"
                  value={formData.name}
                  onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                />
              </Box>
              <Box width="full">
                <Text fontSize="sm" fontWeight="medium" mb={1}>
                  Query
                </Text>
                <Input
                  placeholder="e.g., user entity:message"
                  value={formData.query}
                  onChange={(e) => setFormData({ ...formData, query: e.target.value })}
                />
              </Box>
              <Box width="full">
                <Text fontSize="sm" fontWeight="medium" mb={1}>
                  Description (optional)
                </Text>
                <Textarea
                  placeholder="Add notes about this search..."
                  value={formData.description}
                  onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                  rows={3}
                />
              </Box>
            </VStack>
          </ModalBody>
          <ModalFooter>
            <Button variant="ghost" mr={3} onClick={onClose}>
              Cancel
            </Button>
            <Button
              colorScheme="blue"
              onClick={handleSave}
              isDisabled={!formData.name || !formData.query}
            >
              {editingSearch ? 'Update' : 'Save'}
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>
    </Box>
  );
};
