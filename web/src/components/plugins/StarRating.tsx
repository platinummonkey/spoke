import { useState } from 'react';
import './StarRating.css';

interface StarRatingProps {
  rating: number;
  maxRating?: number;
  size?: 'small' | 'medium' | 'large';
  interactive?: boolean;
  onRatingChange?: (rating: number) => void;
}

export default function StarRating({
  rating,
  maxRating = 5,
  size = 'small',
  interactive = false,
  onRatingChange,
}: StarRatingProps) {
  const [hoveredRating, setHoveredRating] = useState<number | null>(null);

  const getStarClass = (starIndex: number): string => {
    const currentRating = hoveredRating !== null ? hoveredRating : rating;

    if (starIndex <= Math.floor(currentRating)) {
      return 'star filled';
    } else if (starIndex === Math.ceil(currentRating) && currentRating % 1 !== 0) {
      return 'star half-filled';
    } else {
      return 'star empty';
    }
  };

  const handleClick = (starIndex: number) => {
    if (interactive && onRatingChange) {
      onRatingChange(starIndex);
    }
  };

  const handleMouseEnter = (starIndex: number) => {
    if (interactive) {
      setHoveredRating(starIndex);
    }
  };

  const handleMouseLeave = () => {
    if (interactive) {
      setHoveredRating(null);
    }
  };

  return (
    <div className={`star-rating ${size} ${interactive ? 'interactive' : ''}`}>
      {Array.from({ length: maxRating }, (_, index) => {
        const starIndex = index + 1;
        return (
          <span
            key={starIndex}
            className={getStarClass(starIndex)}
            onClick={() => handleClick(starIndex)}
            onMouseEnter={() => handleMouseEnter(starIndex)}
            onMouseLeave={handleMouseLeave}
            role={interactive ? 'button' : undefined}
            aria-label={`${starIndex} stars`}
          >
            ★
          </span>
        );
      })}
      <span className="rating-value">{rating.toFixed(1)}</span>
    </div>
  );
}
