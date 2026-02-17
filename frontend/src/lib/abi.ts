export const VAULT_GATE_ABI = [
  "function payToUnlock() payable",
  "function storeMessage(string message)",
  "function getMessageCount(address user) view returns (uint256)",
  "function getLastMessage(address user) view returns (string)",
  "function getGlobalMessageCount() view returns (uint256)",
  "function getGlobalLastMessage() view returns (string)",
];

export const MIRROR_NFT_ABI = [
  "function mint(uint256 tokenId, string uri)",
  "function transferFrom(address from, address to, uint256 tokenId)",
  "function ownerOf(uint256 tokenId) view returns (address owner, string ownerCosmos, bool exists)",
  "function balanceOf(address owner) view returns (uint256)",
  "function tokenURI(uint256 tokenId) view returns (string)",
  "function tokensOfOwner(address owner) view returns (uint256[])",
  "function exists(uint256 tokenId) view returns (bool)",
  "function getNFT(uint256 tokenId) view returns (address owner, string ownerCosmos, string uri, bool exists)",
];
