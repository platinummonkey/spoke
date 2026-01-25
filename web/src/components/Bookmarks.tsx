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
  Badge,
  Menu,
  MenuButton,
  MenuList,
  MenuItem,
  useToast,
  Divider,
  Tag,
  TagLabel,
  TagCloseButton,
  Wrap,
  WrapItem,
  Link,
} from '@chakra-ui/react';
import { StarIcon, DeleteIcon, EditIcon, ExternalLinkIcon, ChevronDownIcon } from '@chakra-ui/icons';
import { Link as RouterLink } from 'react-router-dom';
import { useBookmarks, Bookmark } from '../hooks/useBookmarks';

export const Bookmarks: React.FC = () => {
  const { bookmarks, loading, updateBookmark, deleteBookmark } = useBookmarks();
  const { isOpen, onOpen, onClose } = useDisclosure();
  const [editingBookmark, setEditingBookmark] = useState<Bookmark | null>(null);
  const [formData, setFormData] = useState({ notes: '', tags: [] as string[], tagInput: '' });
  const toast = useToast();

  const handleEdit = (bookmark: Bookmark) => {
    setEditingBookmark(bookmark);
    setFormData({
      notes: bookmark.notes || '',
      tags: bookmark.tags || [],
      tagInput: '',
    });
    onOpen();
  };

  const handleSave = async () => {
    if (!editingBookmark) return;

    try {
      await updateBookmark(editingBookmark.id, {
        notes: formData.notes,
        tags: formData.tags,
      });
      toast({
        title: 'Bookmark updated',
        status: 'success',
        duration: 2000,
      });
      onClose();
    } catch (err) {
      toast({
        title: 'Error',
        description: 'Failed to update bookmark',
        status: 'error',
        duration: 3000,
      });
    }
  };

  const handleDelete = async (id: number, moduleName: string) => {
    if (window.confirm(`Remove bookmark for "${moduleName}"?`)) {
      try {
        await deleteBookmark(id);
        toast({
          title: 'Bookmark removed',
          status: 'success',
          duration: 2000,
        });
      } catch (err) {
        toast({
          title: 'Error',
          description: 'Failed to delete bookmark',
          status: 'error',
          duration: 3000,
        });
      }
    }
  };

  const handleAddTag = () => {
    if (formData.tagInput.trim() && !formData.tags.includes(formData.tagInput.trim())) {
      setFormData({
        ...formData,
        tags: [...formData.tags, formData.tagInput.trim()],
        tagInput: '',
      });
    }
  };

  const handleRemoveTag = (tag: string) => {
    setFormData({
      ...formData,
      tags: formData.tags.filter(t => t !== tag),
    });
  };

  if (loading) {
    return (
      <Box p={4}>
        <Text fontSize="sm" color="gray.600">
          Loading bookmarks...
        </Text>
      </Box>
    );
  }

  return (
    <Box>
      <HStack justify="space-between" mb={4}>
        <Text fontWeight="bold" fontSize="md">
          Bookmarks
        </Text>
        <Badge colorScheme="blue">{bookmarks.length}</Badge>
      </HStack>

      {bookmarks.length === 0 ? (
        <Box p={4} bg="gray.50" borderRadius="md">
          <Text fontSize="sm" color="gray.600" textAlign="center">
            No bookmarks yet
          </Text>
          <Text fontSize="xs" color="gray.500" mt={2} textAlign="center">
            Click the star icon on any module to bookmark it
          </Text>
        </Box>
      ) : (
        <VStack align="stretch" spacing={2}>
          {bookmarks.map((bookmark) => (
            <Box
              key={bookmark.id}
              p={3}
              bg="white"
              borderRadius="md"
              borderWidth={1}
              borderColor="gray.200"
              _hover={{ borderColor: 'orange.300', bg: 'orange.50' }}
              transition="all 0.2s"
            >
              <HStack justify="space-between" align="start">
                <VStack align="stretch" spacing={1} flex={1}>
                  <HStack>
                    <StarIcon color="orange.400" boxSize={3} />
                    <Link
                      as={RouterLink}
                      to={`/modules/${bookmark.module_name}?version=${bookmark.version}`}
                      fontWeight="medium"
                      fontSize="sm"
                      color="blue.600"
                      _hover={{ textDecoration: 'underline' }}
                    >
                      {bookmark.module_name}
                    </Link>
                  </HStack>
                  <HStack spacing={2}>
                    <Badge colorScheme="gray" fontSize="xs">
                      {bookmark.version}
                    </Badge>
                    {bookmark.entity_type && (
                      <Badge colorScheme="purple" fontSize="xs">
                        {bookmark.entity_type}
                      </Badge>
                    )}
                  </HStack>
                  {bookmark.entity_path && (
                    <Text fontSize="xs" color="gray.600">
                      {bookmark.entity_path}
                    </Text>
                  )}
                  {bookmark.notes && (
                    <Text fontSize="xs" color="gray.500" noOfLines={2}>
                      {bookmark.notes}
                    </Text>
                  )}
                  {bookmark.tags && bookmark.tags.length > 0 && (
                    <Wrap spacing={1} mt={1}>
                      {bookmark.tags.map((tag) => (
                        <WrapItem key={tag}>
                          <Tag size="sm" colorScheme="blue" variant="subtle">
                            {tag}
                          </Tag>
                        </WrapItem>
                      ))}
                    </Wrap>
                  )}
                </VStack>
                <Menu>
                  <MenuButton
                    as={IconButton}
                    icon={<ChevronDownIcon />}
                    size="xs"
                    variant="ghost"
                    aria-label="Bookmark options"
                  />
                  <MenuList fontSize="sm">
                    <MenuItem
                      icon={<ExternalLinkIcon />}
                      as={RouterLink}
                      to={`/modules/${bookmark.module_name}?version=${bookmark.version}`}
                    >
                      Open Module
                    </MenuItem>
                    <MenuItem icon={<EditIcon />} onClick={() => handleEdit(bookmark)}>
                      Edit Notes & Tags
                    </MenuItem>
                    <Divider />
                    <MenuItem
                      icon={<DeleteIcon />}
                      onClick={() => handleDelete(bookmark.id, bookmark.module_name)}
                      color="red.600"
                    >
                      Remove Bookmark
                    </MenuItem>
                  </MenuList>
                </Menu>
              </HStack>
            </Box>
          ))}
        </VStack>
      )}

      {/* Edit Modal */}
      <Modal isOpen={isOpen} onClose={onClose}>
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>Edit Bookmark</ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            <VStack spacing={4}>
              {editingBookmark && (
                <Box width="full" p={3} bg="gray.50" borderRadius="md">
                  <HStack>
                    <StarIcon color="orange.400" boxSize={3} />
                    <Text fontWeight="bold" fontSize="sm">
                      {editingBookmark.module_name}
                    </Text>
                    <Badge colorScheme="gray" fontSize="xs">
                      {editingBookmark.version}
                    </Badge>
                  </HStack>
                  {editingBookmark.entity_path && (
                    <Text fontSize="xs" color="gray.600" mt={1}>
                      {editingBookmark.entity_path}
                    </Text>
                  )}
                </Box>
              )}

              <Box width="full">
                <Text fontSize="sm" fontWeight="medium" mb={1}>
                  Notes
                </Text>
                <Textarea
                  placeholder="Add notes about this bookmark..."
                  value={formData.notes}
                  onChange={(e) => setFormData({ ...formData, notes: e.target.value })}
                  rows={4}
                />
              </Box>

              <Box width="full">
                <Text fontSize="sm" fontWeight="medium" mb={1}>
                  Tags
                </Text>
                <HStack>
                  <Input
                    placeholder="Add tag..."
                    value={formData.tagInput}
                    onChange={(e) => setFormData({ ...formData, tagInput: e.target.value })}
                    onKeyPress={(e) => e.key === 'Enter' && handleAddTag()}
                    size="sm"
                  />
                  <Button size="sm" onClick={handleAddTag}>
                    Add
                  </Button>
                </HStack>
                {formData.tags.length > 0 && (
                  <Wrap spacing={2} mt={2}>
                    {formData.tags.map((tag) => (
                      <WrapItem key={tag}>
                        <Tag size="sm" colorScheme="blue">
                          <TagLabel>{tag}</TagLabel>
                          <TagCloseButton onClick={() => handleRemoveTag(tag)} />
                        </Tag>
                      </WrapItem>
                    ))}
                  </Wrap>
                )}
              </Box>
            </VStack>
          </ModalBody>
          <ModalFooter>
            <Button variant="ghost" mr={3} onClick={onClose}>
              Cancel
            </Button>
            <Button colorScheme="blue" onClick={handleSave}>
              Save
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>
    </Box>
  );
};
