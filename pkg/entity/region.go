/**
 * @Author: Zachariah Knight <zach>
 * @Date:   08-21-2019
 * @Email:  aeros.storkpk@gmail.com
 * @Project: RSCGo
 * @Last modified by:   zach
 * @Last modified time: 08-22-2019
 * @License: Use of this source code is governed by the MIT license that can be found in the LICENSE file.
 * @Copyright: Copyright (c) 2019 Zachariah Knight <aeros.storkpk@gmail.com>
 */

package entity

import "fmt"

const (
	//RegionSize Represents the size of the region
	RegionSize = 48
	//HorizontalPlanes Represents how many columns of regions there are
	HorizontalPlanes = MaxX/RegionSize + 1
	//VerticalPlanes Represents how many rows of regions there are
	VerticalPlanes = MaxY/RegionSize + 1
	//LowerBound Represents a dividing line in the exact middle of a region
	LowerBound = RegionSize / 2
)

//Region Represents a 48x48 section of map.  The purpose of this is to keep track of entities in the entire world without having to allocate tiles individually, which would make search algorithms slower and utilizes a great deal of memory.
type Region struct {
	Players []*Player
	Objects []*Object
}

var regions [HorizontalPlanes][VerticalPlanes]*Region

//AddPlayer Add a player to the region.
func (r *Region) AddPlayer(p *Player) {
	r.Players = append(r.Players, p)
}

//RemovePlayer Remove a player from the region.
func (r *Region) RemovePlayer(p *Player) {
	players := r.Players
	for i, v := range players {
		if v.Index == p.Index {
			last := len(players) - 1
			players[i] = players[last]
			r.Players = players[:last]
			return
		}
	}
}

//AddObject Add an object to the region.
func (r *Region) AddObject(o *Object) {
	r.Objects = append(r.Objects, o)
}

//RemoveObject Remove an object from the region.
func (r *Region) RemoveObject(o *Object) {
	objects := r.Objects
	for i, v := range objects {
		if v.Index == o.Index {
			last := len(objects) - 1
			objects[i] = objects[last]
			r.Objects = objects[:last]
			return
		}
	}
}

//getRegionFromIndex internal function to get a region by its row amd column indexes
func getRegionFromIndex(areaX, areaY int) *Region {
	if areaX < 0 || areaX >= HorizontalPlanes {
		fmt.Println("planeX index out of range")
		return &Region{}
	}
	if areaY < 0 || areaY >= VerticalPlanes {
		fmt.Println("planeY index out of range")
		return &Region{}
	}
	if regions[areaX][areaY] == nil {
		regions[areaX][areaY] = &Region{}
	}
	return regions[areaX][areaY]
}

//GetRegion Returns the region that corresponds with the given coordinates.  If it does not exist yet, it will allocate a new onr and store it for the lifetime of the application in the regions map.
func GetRegion(x, y int) *Region {
	return getRegionFromIndex(x/RegionSize, y/RegionSize)
}

//GetRegionFromLocation Returns the region that corresponds with the given location.  If it does not exist yet, it will allocate a new onr and store it for the lifetime of the application in the regions map.
func GetRegionFromLocation(loc *Location) *Region {
	return getRegionFromIndex(loc.X/RegionSize, loc.Y/RegionSize)
}

//SurroundingRegions Returns the regions surrounding the given coordinates.  It wil
func SurroundingRegions(x, y int) (regions [4]*Region) {
	areaX := x / RegionSize
	areaY := y / RegionSize
	regions[0] = getRegionFromIndex(areaX, areaY)
	relX := x % RegionSize
	relY := y % RegionSize
	if relX <= LowerBound {
		if relY <= LowerBound {
			regions[1] = getRegionFromIndex(areaX-1, areaY)
			regions[2] = getRegionFromIndex(areaX-1, areaY-1)
			regions[3] = getRegionFromIndex(areaX, areaY-1)
		} else {
			regions[1] = getRegionFromIndex(areaX-1, areaY)
			regions[2] = getRegionFromIndex(areaX-1, areaY+1)
			regions[3] = getRegionFromIndex(areaX, areaY+1)
		}
	} else if relY <= LowerBound {
		regions[1] = getRegionFromIndex(areaX+1, areaY)
		regions[2] = getRegionFromIndex(areaX+1, areaY-1)
		regions[3] = getRegionFromIndex(areaX, areaY-1)
	} else {
		regions[1] = getRegionFromIndex(areaX+1, areaY)
		regions[2] = getRegionFromIndex(areaX+1, areaY+1)
		regions[3] = getRegionFromIndex(areaX, areaY+1)
	}

	return
}
