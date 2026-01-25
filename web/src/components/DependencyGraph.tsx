import React, { useEffect, useRef, useState } from 'react';
import {
  Box,
  Button,
  ButtonGroup,
  Flex,
  Heading,
  IconButton,
  Select,
  Tooltip,
  VStack,
  HStack,
  Badge,
  Alert,
  AlertIcon,
  AlertTitle,
  AlertDescription,
  useToast,
} from '@chakra-ui/react';
import { DownloadIcon, RepeatIcon } from '@chakra-ui/icons';
import cytoscape, { Core, NodeSingular, EventObject } from 'cytoscape';
// @ts-ignore - No types available for these plugins
import cola from 'cytoscape-cola';
// @ts-ignore
import dagre from 'cytoscape-dagre';

// Register layout extensions
cytoscape.use(cola);
cytoscape.use(dagre);

interface DependencyGraphProps {
  moduleName: string;
  version: string;
  transitive?: boolean;
  direction?: 'dependencies' | 'dependents' | 'both';
  maxDepth?: number;
  onNodeClick?: (module: string, version: string) => void;
}

interface CytoscapeNode {
  data: {
    id: string;
    name: string;
    version: string;
    type: 'current' | 'dependency' | 'dependent';
  };
}

interface CytoscapeEdge {
  data: {
    id: string;
    source: string;
    target: string;
    type?: string;
  };
}

interface CytoscapeGraphData {
  nodes: CytoscapeNode[];
  edges: CytoscapeEdge[];
}

type LayoutType = 'dagre' | 'cola' | 'breadthfirst' | 'circle' | 'grid';

const DependencyGraph: React.FC<DependencyGraphProps> = ({
  moduleName,
  version,
  transitive = true,
  direction = 'dependencies',
  maxDepth,
  onNodeClick,
}) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const cyRef = useRef<Core | null>(null);
  const [layout, setLayout] = useState<LayoutType>('dagre');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [graphData, setGraphData] = useState<CytoscapeGraphData | null>(null);
  const toast = useToast();

  // Fetch graph data
  useEffect(() => {
    const fetchGraphData = async () => {
      setLoading(true);
      setError(null);

      try {
        const params = new URLSearchParams({
          transitive: String(transitive),
          direction,
        });

        if (maxDepth !== undefined) {
          params.append('depth', String(maxDepth));
        }

        const response = await fetch(
          `/api/v2/modules/${moduleName}/versions/${version}/graph?${params}`
        );

        if (!response.ok) {
          throw new Error(`Failed to fetch graph: ${response.statusText}`);
        }

        const data: CytoscapeGraphData = await response.json();
        setGraphData(data);
      } catch (err) {
        const message = err instanceof Error ? err.message : 'Unknown error';
        setError(message);
        console.error('Failed to fetch dependency graph:', err);
      } finally {
        setLoading(false);
      }
    };

    fetchGraphData();
  }, [moduleName, version, transitive, direction, maxDepth]);

  // Initialize and render Cytoscape graph
  useEffect(() => {
    if (!containerRef.current || !graphData || loading) {
      return;
    }

    // Destroy existing instance
    if (cyRef.current) {
      cyRef.current.destroy();
    }

    // Create Cytoscape instance
    const cy = cytoscape({
      container: containerRef.current,
      elements: {
        nodes: graphData.nodes.map((node) => ({
          data: node.data,
        })),
        edges: graphData.edges.map((edge) => ({
          data: edge.data,
        })),
      },
      style: [
        {
          selector: 'node',
          style: {
            label: 'data(name)',
            'text-valign': 'center',
            'text-halign': 'center',
            'background-color': '#cbd5e0',
            width: '60px',
            height: '60px',
            'font-size': '12px',
            'text-wrap': 'wrap',
            'text-max-width': '80px',
          },
        },
        {
          selector: 'node[type="current"]',
          style: {
            'background-color': '#3182ce',
            color: '#fff',
            'font-weight': 'bold',
            width: '80px',
            height: '80px',
          },
        },
        {
          selector: 'node[type="dependency"]',
          style: {
            'background-color': '#48bb78',
          },
        },
        {
          selector: 'node[type="dependent"]',
          style: {
            'background-color': '#ed8936',
          },
        },
        {
          selector: 'edge',
          style: {
            width: 2,
            'line-color': '#a0aec0',
            'target-arrow-color': '#a0aec0',
            'target-arrow-shape': 'triangle',
            'curve-style': 'bezier',
            label: 'data(type)',
            'font-size': '10px',
            'text-rotation': 'autorotate',
            'text-margin-y': -10,
          },
        },
        {
          selector: 'edge[type="direct"]',
          style: {
            width: 3,
            'line-color': '#2d3748',
          },
        },
        {
          selector: 'edge[type="transitive"]',
          style: {
            'line-style': 'dashed',
          },
        },
      ],
      layout: getLayoutConfig(layout),
      minZoom: 0.5,
      maxZoom: 2,
    });

    // Add click handler for nodes
    cy.on('tap', 'node', (evt: EventObject) => {
      const node: NodeSingular = evt.target;
      const nodeData = node.data();

      if (onNodeClick && nodeData.name && nodeData.version) {
        onNodeClick(nodeData.name, nodeData.version);
      }
    });

    // Add hover effects
    cy.on('mouseover', 'node', (evt: EventObject) => {
      evt.target.style('cursor', 'pointer');
    });

    cyRef.current = cy;

    return () => {
      if (cyRef.current) {
        cyRef.current.destroy();
      }
    };
  }, [graphData, layout, loading, onNodeClick]);

  // Change layout
  const changeLayout = (newLayout: LayoutType) => {
    setLayout(newLayout);

    if (cyRef.current) {
      const layoutConfig = getLayoutConfig(newLayout);
      cyRef.current.layout(layoutConfig).run();
    }
  };

  // Export as PNG
  const exportPNG = () => {
    if (!cyRef.current) return;

    const png = cyRef.current.png({ output: 'blob', scale: 2 });
    const url = URL.createObjectURL(png);
    const link = document.createElement('a');
    link.href = url;
    link.download = `${moduleName}-${version}-dependencies.png`;
    link.click();
    URL.revokeObjectURL(url);

    toast({
      title: 'Graph exported',
      description: 'Dependency graph saved as PNG',
      status: 'success',
      duration: 3000,
    });
  };

  // Export as JSON
  const exportJSON = () => {
    if (!graphData) return;

    const json = JSON.stringify(graphData, null, 2);
    const blob = new Blob([json], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = `${moduleName}-${version}-dependencies.json`;
    link.click();
    URL.revokeObjectURL(url);

    toast({
      title: 'Data exported',
      description: 'Dependency data saved as JSON',
      status: 'success',
      duration: 3000,
    });
  };

  // Refresh graph
  const refreshGraph = () => {
    window.location.reload();
  };

  if (error) {
    return (
      <Alert status="error">
        <AlertIcon />
        <AlertTitle>Failed to load dependency graph</AlertTitle>
        <AlertDescription>{error}</AlertDescription>
      </Alert>
    );
  }

  return (
    <VStack spacing={4} align="stretch" height="100%">
      {/* Header */}
      <Flex justify="space-between" align="center">
        <Heading size="md">Dependency Graph</Heading>
        <HStack spacing={2}>
          {/* Layout selector */}
          <Select
            size="sm"
            value={layout}
            onChange={(e) => changeLayout(e.target.value as LayoutType)}
            width="150px"
          >
            <option value="dagre">Hierarchical (Dagre)</option>
            <option value="cola">Force-Directed (Cola)</option>
            <option value="breadthfirst">Breadth-First</option>
            <option value="circle">Circle</option>
            <option value="grid">Grid</option>
          </Select>

          {/* Export buttons */}
          <ButtonGroup size="sm" isAttached>
            <Tooltip label="Export as PNG">
              <IconButton
                aria-label="Export PNG"
                icon={<DownloadIcon />}
                onClick={exportPNG}
                isDisabled={!graphData}
              />
            </Tooltip>
            <Tooltip label="Export as JSON">
              <Button onClick={exportJSON} isDisabled={!graphData}>
                JSON
              </Button>
            </Tooltip>
          </ButtonGroup>

          {/* Refresh button */}
          <Tooltip label="Refresh graph">
            <IconButton
              aria-label="Refresh"
              icon={<RepeatIcon />}
              onClick={refreshGraph}
              size="sm"
            />
          </Tooltip>
        </HStack>
      </Flex>

      {/* Legend */}
      <HStack spacing={4}>
        <HStack>
          <Badge colorScheme="blue">Current Module</Badge>
          <Badge colorScheme="green">Dependencies</Badge>
          <Badge colorScheme="orange">Dependents</Badge>
        </HStack>
        <HStack fontSize="sm" color="gray.600">
          <Box>Solid line = Direct</Box>
          <Box>Dashed line = Transitive</Box>
        </HStack>
      </HStack>

      {/* Graph container */}
      <Box
        ref={containerRef}
        flex="1"
        border="1px solid"
        borderColor="gray.200"
        borderRadius="md"
        bg="white"
        minHeight="500px"
        position="relative"
      >
        {loading && (
          <Flex
            position="absolute"
            top="0"
            left="0"
            right="0"
            bottom="0"
            align="center"
            justify="center"
            bg="rgba(255, 255, 255, 0.8)"
            zIndex={10}
          >
            Loading graph...
          </Flex>
        )}
      </Box>

      {/* Stats */}
      {graphData && (
        <Flex justify="space-between" fontSize="sm" color="gray.600">
          <Box>Nodes: {graphData.nodes.length}</Box>
          <Box>Edges: {graphData.edges.length}</Box>
        </Flex>
      )}
    </VStack>
  );
};

// Layout configurations
function getLayoutConfig(layout: LayoutType) {
  switch (layout) {
    case 'dagre':
      return {
        name: 'dagre',
        rankDir: 'TB', // Top to bottom
        nodeSep: 50,
        edgeSep: 10,
        rankSep: 100,
      };
    case 'cola':
      return {
        name: 'cola',
        nodeSpacing: 50,
        edgeLength: 100,
        animate: true,
        randomize: false,
        maxSimulationTime: 2000,
      };
    case 'breadthfirst':
      return {
        name: 'breadthfirst',
        directed: true,
        spacingFactor: 1.5,
      };
    case 'circle':
      return {
        name: 'circle',
        spacingFactor: 1.5,
      };
    case 'grid':
      return {
        name: 'grid',
        rows: undefined,
        cols: undefined,
      };
    default:
      return { name: 'dagre' };
  }
}

export default DependencyGraph;
